package http

import (
	"bytes"
	"errors"
	"github.com/newacorn/goutils/unsafefn"
	"io"
	"path/filepath"
	"sync"
	"time"
)

var zeroTime time.Time

var (
	// CookieExpireDelete may be set on Cookie.Expire for expiring the given cookie.
	CookieExpireDelete = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	// CookieExpireUnlimited indicates that the cookie doesn't expire.
	CookieExpireUnlimited = zeroTime
)

// CookieSameSite is an enum for the mode in which the SameSite flag should be set for the given cookie.
// See https://tools.ietf.org/html/draft-ietf-httpbis-cookie-same-site-00 for details.
type CookieSameSite int

const (
	// CookieSameSiteDisabled removes the SameSite flag.
	CookieSameSiteDisabled CookieSameSite = iota
	// CookieSameSiteDefaultMode sets the SameSite flag.
	CookieSameSiteDefaultMode
	// CookieSameSiteLaxMode sets the SameSite flag with the "Lax" parameter.
	CookieSameSiteLaxMode
	// CookieSameSiteStrictMode sets the SameSite flag with the "Strict" parameter.
	CookieSameSiteStrictMode
	// CookieSameSiteNoneMode sets the SameSite flag with the "None" parameter.
	// See https://tools.ietf.org/html/draft-west-cookie-incrementalism-00
	CookieSameSiteNoneMode // third-party cookies are phasing out, use Partitioned cookies instead
)

// AcquireCookie returns an empty Cookie object from the pool.
//
// The returned object may be returned back to the pool with ReleaseCookie.
// This allows reducing GC load.
func AcquireCookie() *Cookie {
	return cookiePool.Get().(*Cookie)
}

// ReleaseCookie returns the Cookie object acquired with AcquireCookie back
// to the pool.
//
// Do not access released Cookie object, otherwise data races may occur.
func ReleaseCookie(c *Cookie) {
	c.Reset()
	cookiePool.Put(c)
}

var cookiePool = &sync.Pool{
	New: func() any {
		return &Cookie{}
	},
}

// Cookie represents HTTP response cookie.
//
// Do not copy Cookie objects. Create new object and use CopyTo instead.
//
// Cookie instance MUST NOT be used from concurrently running goroutines.
type Cookie struct {
	noCopy noCopy

	key    []byte
	value  []byte
	expire time.Time
	maxAge int
	domain []byte
	path   []byte

	sameSite    CookieSameSite
	httpOnly    bool
	secure      bool
	partitioned bool

	bufKV argsKV
	buf   []byte
}

// CopyTo copies src cookie to c.
func (c *Cookie) CopyTo(src *Cookie) {
	c.Reset()
	c.key = append(c.key, src.key...)
	c.value = append(c.value, src.value...)
	c.expire = src.expire
	c.maxAge = src.maxAge
	c.domain = append(c.domain, src.domain...)
	c.path = append(c.path, src.path...)
	c.httpOnly = src.httpOnly
	c.secure = src.secure
	c.sameSite = src.sameSite
	c.partitioned = src.partitioned
}

// HTTPOnly returns true if the cookie is http only.
func (c *Cookie) HTTPOnly() bool {
	return c.httpOnly
}

// SetHTTPOnly sets cookie's httpOnly flag to the given value.
func (c *Cookie) SetHTTPOnly(httpOnly bool) {
	c.httpOnly = httpOnly
}

// Secure returns true if the cookie is secure.
func (c *Cookie) Secure() bool {
	return c.secure
}

// SetSecure sets cookie's secure flag to the given value.
func (c *Cookie) SetSecure(secure bool) {
	c.secure = secure
}

// SameSite returns the SameSite mode.
func (c *Cookie) SameSite() CookieSameSite {
	return c.sameSite
}

// SetSameSite sets the cookie's SameSite flag to the given value.
// Set value CookieSameSiteNoneMode will set Secure to true also to avoid browser rejection.
func (c *Cookie) SetSameSite(mode CookieSameSite) {
	c.sameSite = mode
	if mode == CookieSameSiteNoneMode {
		c.SetSecure(true)
	}
}

// Partitioned returns true if the cookie is partitioned.
func (c *Cookie) Partitioned() bool {
	return c.partitioned
}

// SetPartitioned sets the cookie's Partitioned flag to the given value.
// Set value Partitioned to true will set Secure to true and Path to / also to avoid browser rejection.
func (c *Cookie) SetPartitioned(partitioned bool) {
	c.partitioned = partitioned
	if partitioned {
		c.SetSecure(true)
		c.SetPath("/")
	}
}

// Path returns cookie path.
func (c *Cookie) Path() []byte {
	return c.path
}

// SetPath sets cookie path.
func (c *Cookie) SetPath(path string) {
	c.buf = append(c.buf[:0], path...)
	c.path = normalizePath(c.path, c.buf)
}

// SetPathBytes sets cookie path.
func (c *Cookie) SetPathBytes(path []byte) {
	c.buf = append(c.buf[:0], path...)
	c.path = normalizePath(c.path, c.buf)
}

// Domain returns cookie domain.
//
// The returned value is valid until the Cookie reused or released (ReleaseCookie).
// Do not store references to the returned value. Make copies instead.
func (c *Cookie) Domain() []byte {
	return c.domain
}

// SetDomain sets cookie domain.
func (c *Cookie) SetDomain(domain string) {
	c.domain = append(c.domain[:0], domain...)
}

// SetDomainBytes sets cookie domain.
func (c *Cookie) SetDomainBytes(domain []byte) {
	c.domain = append(c.domain[:0], domain...)
}

// MaxAge returns the seconds until the cookie is meant to expire or 0
// if no max age.
func (c *Cookie) MaxAge() int {
	return c.maxAge
}

// SetMaxAge sets cookie expiration time based on seconds. This takes precedence
// over any absolute expiry set on the cookie.
//
// Set max age to 0 to unset.
func (c *Cookie) SetMaxAge(seconds int) {
	c.maxAge = seconds
}

// Expire returns cookie expiration time.
//
// CookieExpireUnlimited is returned if cookie doesn't expire.
func (c *Cookie) Expire() time.Time {
	expire := c.expire
	if expire.IsZero() {
		expire = CookieExpireUnlimited
	}
	return expire
}

// SetExpire sets cookie expiration time.
//
// Set expiration time to CookieExpireDelete for expiring (deleting)
// the cookie on the client.
//
// By default cookie lifetime is limited by browser session.
func (c *Cookie) SetExpire(expire time.Time) {
	c.expire = expire
}

// Value returns cookie value.
//
// The returned value is valid until the Cookie reused or released (ReleaseCookie).
// Do not store references to the returned value. Make copies instead.
func (c *Cookie) Value() []byte {
	return c.value
}

// SetValue sets cookie value.
func (c *Cookie) SetValue(value string) {
	c.value = append(c.value[:0], value...)
}

// SetValueBytes sets cookie value.
func (c *Cookie) SetValueBytes(value []byte) {
	c.value = append(c.value[:0], value...)
}

// Key returns cookie name.
//
// The returned value is valid until the Cookie reused or released (ReleaseCookie).
// Do not store references to the returned value. Make copies instead.
func (c *Cookie) Key() []byte {
	return c.key
}

// SetKey sets cookie name.
func (c *Cookie) SetKey(key string) {
	c.key = append(c.key[:0], key...)
}

// SetKeyBytes sets cookie name.
func (c *Cookie) SetKeyBytes(key []byte) {
	c.key = append(c.key[:0], key...)
}

// Reset clears the cookie.
func (c *Cookie) Reset() {
	c.key = c.key[:0]
	c.value = c.value[:0]
	c.expire = zeroTime
	c.maxAge = 0
	c.domain = c.domain[:0]
	c.path = c.path[:0]
	c.httpOnly = false
	c.secure = false
	c.sameSite = CookieSameSiteDisabled
	c.partitioned = false
}

// AppendBytes appends cookie representation to dst and returns
// the extended dst.
func (c *Cookie) AppendBytes(dst []byte) []byte {
	if len(c.key) > 0 {
		dst = append(dst, c.key...)
		dst = append(dst, '=')
	}
	dst = append(dst, c.value...)

	if c.maxAge > 0 {
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieMaxAge...)
		dst = append(dst, '=')
		dst = AppendUint(dst, c.maxAge)
	} else if !c.expire.IsZero() {
		c.bufKV.value = AppendHTTPDate(c.bufKV.value[:0], c.expire)
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieExpires...)
		dst = append(dst, '=')
		dst = append(dst, c.bufKV.value...)
	}
	if len(c.domain) > 0 {
		dst = appendCookiePart(dst, strCookieDomain, c.domain)
	}
	if len(c.path) > 0 {
		dst = appendCookiePart(dst, strCookiePath, c.path)
	}
	if c.httpOnly {
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieHTTPOnly...)
	}
	if c.secure {
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieSecure...)
	}
	switch c.sameSite {
	case CookieSameSiteDefaultMode:
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieSameSite...)
	case CookieSameSiteLaxMode:
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieSameSite...)
		dst = append(dst, '=')
		dst = append(dst, strCookieSameSiteLax...)
	case CookieSameSiteStrictMode:
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieSameSite...)
		dst = append(dst, '=')
		dst = append(dst, strCookieSameSiteStrict...)
	case CookieSameSiteNoneMode:
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookieSameSite...)
		dst = append(dst, '=')
		dst = append(dst, strCookieSameSiteNone...)
	}
	if c.partitioned {
		dst = append(dst, ';', ' ')
		dst = append(dst, strCookiePartitioned...)
	}
	return dst
}

// Cookie returns cookie representation.
//
// The returned value is valid until the Cookie reused or released (ReleaseCookie).
// Do not store references to the returned value. Make copies instead.
func (c *Cookie) Cookie() []byte {
	c.buf = c.AppendBytes(c.buf[:0])
	return c.buf
}

// String returns cookie representation.
func (c *Cookie) String() string {
	return string(c.Cookie())
}

// WriteTo writes cookie representation to w.
//
// WriteTo implements io.WriterTo interface.
func (c *Cookie) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(c.Cookie())
	return int64(n), err
}

var errNoCookies = errors.New("no cookies found")

// Parse parses Set-Cookie header.
func (c *Cookie) Parse(src string) error {
	c.buf = append(c.buf[:0], src...)
	return c.ParseBytes(c.buf)
}

// ParseBytes parses Set-Cookie header.
func (c *Cookie) ParseBytes(src []byte) error {
	c.Reset()

	var s cookieScanner
	s.b = src

	kv := &c.bufKV
	if !s.next(kv) {
		return errNoCookies
	}

	c.key = append(c.key, kv.key...)
	c.value = append(c.value, kv.value...)

	for s.next(kv) {
		if len(kv.key) != 0 {
			// Case insensitive switch on first char
			switch kv.key[0] | 0x20 {
			case 'm':
				if caseInsensitiveCompare(strCookieMaxAge, kv.key) {
					maxAge, err := ParseUint(kv.value)
					if err != nil {
						return err
					}
					c.maxAge = maxAge
				}

			case 'e': // "expires"
				if caseInsensitiveCompare(strCookieExpires, kv.key) {
					v := unsafefn.B2S(kv.value)
					// Try the same two formats as net/http
					// See: https://github.com/golang/go/blob/00379be17e63a5b75b3237819392d2dc3b313a27/src/net/http/cookie.go#L133-L135
					exptime, err := time.ParseInLocation(time.RFC1123, v, time.UTC)
					if err != nil {
						exptime, err = time.Parse("Mon, 02-Jan-2006 15:04:05 MST", v)
						if err != nil {
							return err
						}
					}
					c.expire = exptime
				}

			case 'd': // "domain"
				if caseInsensitiveCompare(strCookieDomain, kv.key) {
					c.domain = append(c.domain, kv.value...)
				}

			case 'p': // "path"
				if caseInsensitiveCompare(strCookiePath, kv.key) {
					c.path = append(c.path, kv.value...)
				}

			case 's': // "samesite"
				if caseInsensitiveCompare(strCookieSameSite, kv.key) {
					if len(kv.value) > 0 {
						// Case insensitive switch on first char
						switch kv.value[0] | 0x20 {
						case 'l': // "lax"
							if caseInsensitiveCompare(strCookieSameSiteLax, kv.value) {
								c.sameSite = CookieSameSiteLaxMode
							}
						case 's': // "strict"
							if caseInsensitiveCompare(strCookieSameSiteStrict, kv.value) {
								c.sameSite = CookieSameSiteStrictMode
							}
						case 'n': // "none"
							if caseInsensitiveCompare(strCookieSameSiteNone, kv.value) {
								c.sameSite = CookieSameSiteNoneMode
							}
						}
					}
				}
			}
		} else if len(kv.value) != 0 {
			// Case insensitive switch on first char
			switch kv.value[0] | 0x20 {
			case 'h': // "httponly"
				if caseInsensitiveCompare(strCookieHTTPOnly, kv.value) {
					c.httpOnly = true
				}

			case 's': // "secure"
				if caseInsensitiveCompare(strCookieSecure, kv.value) {
					c.secure = true
				} else if caseInsensitiveCompare(strCookieSameSite, kv.value) {
					c.sameSite = CookieSameSiteDefaultMode
				}
			case 'p': // "partitioned"
				if caseInsensitiveCompare(strCookiePartitioned, kv.value) {
					c.partitioned = true
				}
			}
		} // else empty or no match
	}
	return nil
}

func appendCookiePart(dst, key, value []byte) []byte {
	dst = append(dst, ';', ' ')
	dst = append(dst, key...)
	dst = append(dst, '=')
	return append(dst, value...)
}

func getCookieKey(dst, src []byte) []byte {
	n := bytes.IndexByte(src, '=')
	if n >= 0 {
		src = src[:n]
	}
	return decodeCookieArg(dst, src, false)
}

func appendRequestCookieBytes(dst []byte, cookies []argsKV) []byte {
	for i, n := 0, len(cookies); i < n; i++ {
		kv := &cookies[i]
		if len(kv.key) > 0 {
			dst = append(dst, kv.key...)
			dst = append(dst, '=')
		}
		dst = append(dst, kv.value...)
		if i+1 < n {
			dst = append(dst, ';', ' ')
		}
	}
	return dst
}

// For Response we can not use the above function as response cookies
// already contain the key= in the value.
func appendResponseCookieBytes(dst []byte, cookies []argsKV) []byte {
	for i, n := 0, len(cookies); i < n; i++ {
		kv := &cookies[i]
		dst = append(dst, kv.value...)
		if i+1 < n {
			dst = append(dst, ';', ' ')
		}
	}
	return dst
}

func parseRequestCookies(cookies []argsKV, src []byte) []argsKV {
	var s cookieScanner
	s.b = src
	var kv *argsKV
	cookies, kv = allocArg(cookies)
	for s.next(kv) {
		if len(kv.key) > 0 || len(kv.value) > 0 {
			cookies, kv = allocArg(cookies)
		}
	}
	return releaseArg(cookies)
}

type cookieScanner struct {
	b []byte
}

func (s *cookieScanner) next(kv *argsKV) bool {
	b := s.b
	if len(b) == 0 {
		return false
	}

	isKey := true
	k := 0
	for i, c := range b {
		switch c {
		case '=':
			if isKey {
				isKey = false
				kv.key = decodeCookieArg(kv.key, b[:i], false)
				k = i + 1
			}
		case ';':
			if isKey {
				kv.key = kv.key[:0]
			}
			kv.value = decodeCookieArg(kv.value, b[k:i], true)
			s.b = b[i+1:]
			return true
		}
	}

	if isKey {
		kv.key = kv.key[:0]
	}
	kv.value = decodeCookieArg(kv.value, b[k:], true)
	s.b = b[len(b):]
	return true
}

func decodeCookieArg(dst, src []byte, skipQuotes bool) []byte {
	for len(src) > 0 && src[0] == ' ' {
		src = src[1:]
	}
	for len(src) > 0 && src[len(src)-1] == ' ' {
		src = src[:len(src)-1]
	}
	if skipQuotes {
		if len(src) > 1 && src[0] == '"' && src[len(src)-1] == '"' {
			src = src[1 : len(src)-1]
		}
	}
	return append(dst[:0], src...)
}

// caseInsensitiveCompare does a case insensitive equality comparison of
// two []byte. Assumes only letters need to be matched.
func caseInsensitiveCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i]|0x20 != b[i]|0x20 {
			return false
		}
	}
	return true
}

type argsKV struct {
	key     []byte
	value   []byte
	noValue bool
}

// Embed this type into a struct, which mustn't be copied,
// so `go vet` gives a warning if this struct is copied.
//
// See https://github.com/golang/go/issues/8005#issuecomment-190753527 for details.
// and also: https://stackoverflow.com/questions/52494458/nocopy-minimal-example
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

func normalizePath(dst, src []byte) []byte {
	dst = dst[:0]
	dst = addLeadingSlash(dst, src)
	dst = decodeArgAppendNoPlus(dst, src)

	// remove duplicate slashes
	b := dst
	bSize := len(b)
	for {
		n := bytes.Index(b, strSlashSlash)
		if n < 0 {
			break
		}
		b = b[n:]
		copy(b, b[1:])
		b = b[:len(b)-1]
		bSize--
	}
	dst = dst[:bSize]

	// remove /./ parts
	b = dst
	for {
		n := bytes.Index(b, strSlashDotSlash)
		if n < 0 {
			break
		}
		nn := n + len(strSlashDotSlash) - 1
		copy(b[n:], b[nn:])
		b = b[:len(b)-nn+n]
	}

	// remove /foo/../ parts
	for {
		n := bytes.Index(b, strSlashDotDotSlash)
		if n < 0 {
			break
		}
		nn := bytes.LastIndexByte(b[:n], '/')
		if nn < 0 {
			nn = 0
		}
		n += len(strSlashDotDotSlash) - 1
		copy(b[nn:], b[n:])
		b = b[:len(b)-n+nn]
	}

	// remove trailing /foo/..
	n := bytes.LastIndex(b, strSlashDotDot)
	if n >= 0 && n+len(strSlashDotDot) == len(b) {
		nn := bytes.LastIndexByte(b[:n], '/')
		if nn < 0 {
			return append(dst[:0], strSlash...)
		}
		b = b[:nn+1]
	}

	if filepath.Separator == '\\' {
		// remove \.\ parts
		for {
			n := bytes.Index(b, strBackSlashDotBackSlash)
			if n < 0 {
				break
			}
			nn := n + len(strSlashDotSlash) - 1
			copy(b[n:], b[nn:])
			b = b[:len(b)-nn+n]
		}

		// remove /foo/..\ parts
		for {
			n := bytes.Index(b, strSlashDotDotBackSlash)
			if n < 0 {
				break
			}
			nn := bytes.LastIndexByte(b[:n], '/')
			if nn < 0 {
				nn = 0
			}
			nn++
			n += len(strSlashDotDotBackSlash)
			copy(b[nn:], b[n:])
			b = b[:len(b)-n+nn]
		}

		// remove /foo\..\ parts
		for {
			n := bytes.Index(b, strBackSlashDotDotBackSlash)
			if n < 0 {
				break
			}
			nn := bytes.LastIndexByte(b[:n], '/')
			if nn < 0 {
				nn = 0
			}
			n += len(strBackSlashDotDotBackSlash) - 1
			copy(b[nn:], b[n:])
			b = b[:len(b)-n+nn]
		}

		// remove trailing \foo\..
		n := bytes.LastIndex(b, strBackSlashDotDot)
		if n >= 0 && n+len(strSlashDotDot) == len(b) {
			nn := bytes.LastIndexByte(b[:n], '/')
			if nn < 0 {
				return append(dst[:0], strSlash...)
			}
			b = b[:nn+1]
		}
	}

	return b
}

var (
	strCookieExpires        = []byte("expires")
	strCookieDomain         = []byte("domain")
	strCookiePath           = []byte("path")
	strCookieHTTPOnly       = []byte("HttpOnly")
	strCookieSecure         = []byte("secure")
	strCookiePartitioned    = []byte("Partitioned")
	strCookieMaxAge         = []byte("max-age")
	strCookieSameSite       = []byte("SameSite")
	strCookieSameSiteLax    = []byte("Lax")
	strCookieSameSiteStrict = []byte("Strict")
	strCookieSameSiteNone   = []byte("None")
)

func addLeadingSlash(dst, src []byte) []byte {
	// add leading slash for unix paths
	if len(src) == 0 || src[0] != '/' {
		dst = append(dst, '/')
	}

	return dst
}

// decodeArgAppendNoPlus is almost identical to decodeArgAppend, but it doesn't
// substitute '+' with ' '.
//
// The function is copy-pasted from decodeArgAppend due to the performance
// reasons only.
func decodeArgAppendNoPlus(dst, src []byte) []byte {
	idx := bytes.IndexByte(src, '%')
	if idx < 0 {
		// fast path: src doesn't contain encoded chars
		return append(dst, src...)
	}
	dst = append(dst, src[:idx]...)

	// slow path
	for i := idx; i < len(src); i++ {
		c := src[i]
		if c == '%' {
			if i+2 >= len(src) {
				return append(dst, src[i:]...)
			}
			x2 := hex2intTable[src[i+2]]
			x1 := hex2intTable[src[i+1]]
			if x1 == 16 || x2 == 16 {
				dst = append(dst, '%')
			} else {
				dst = append(dst, x1<<4|x2)
				i += 2
			}
		} else {
			dst = append(dst, c)
		}
	}
	return dst
}

var (
	strSlash                    = []byte("/")
	strSlashSlash               = []byte("//")
	strSlashDotDot              = []byte("/..")
	strSlashDotSlash            = []byte("/./")
	strSlashDotDotSlash         = []byte("/../")
	strBackSlashDotDot          = []byte(`\..`)
	strBackSlashDotBackSlash    = []byte(`\.\`)
	strSlashDotDotBackSlash     = []byte(`/..\`)
	strBackSlashDotDotBackSlash = []byte(`\..\`)
	strCRLF                     = []byte("\r\n")
	strHTTP                     = []byte("http")
	strHTTPS                    = []byte("https")
	strHTTP11                   = []byte("HTTP/1.1")
	strColon                    = []byte(":")
	strColonSlashSlash          = []byte("://")
	strColonSpace               = []byte(": ")
	strCommaSpace               = []byte(", ")
	strGMT                      = []byte("GMT")
)

const hex2intTable = "\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x00\x01\x02\x03\x04\x05\x06\a\b\t\x10\x10\x10\x10\x10\x10\x10\n\v\f\r\x0e\x0f\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\n\v\f\r\x0e\x0f\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10"
const toLowerTable = "\x00\x01\x02\x03\x04\x05\x06\a\b\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f !\"#$%&'()*+,-./0123456789:;<=>?@abcdefghijklmnopqrstuvwxyz[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~\x7f\x80\x81\x82\x83\x84\x85\x86\x87\x88\x89\x8a\x8b\x8c\x8d\x8e\x8f\x90\x91\x92\x93\x94\x95\x96\x97\x98\x99\x9a\x9b\x9c\x9d\x9e\x9f\xa0\xa1\xa2\xa3\xa4\xa5\xa6\xa7\xa8\xa9\xaa\xab\xac\xad\xae\xaf\xb0\xb1\xb2\xb3\xb4\xb5\xb6\xb7\xb8\xb9\xba\xbb\xbc\xbd\xbe\xbf\xc0\xc1\xc2\xc3\xc4\xc5\xc6\xc7\xc8\xc9\xca\xcb\xcc\xcd\xce\xcf\xd0\xd1\xd2\xd3\xd4\xd5\xd6\xd7\xd8\xd9\xda\xdb\xdc\xdd\xde\xdf\xe0\xe1\xe2\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff"
const toUpperTable = "\x00\x01\x02\x03\x04\x05\x06\a\b\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`ABCDEFGHIJKLMNOPQRSTUVWXYZ{|}~\x7f\x80\x81\x82\x83\x84\x85\x86\x87\x88\x89\x8a\x8b\x8c\x8d\x8e\x8f\x90\x91\x92\x93\x94\x95\x96\x97\x98\x99\x9a\x9b\x9c\x9d\x9e\x9f\xa0\xa1\xa2\xa3\xa4\xa5\xa6\xa7\xa8\xa9\xaa\xab\xac\xad\xae\xaf\xb0\xb1\xb2\xb3\xb4\xb5\xb6\xb7\xb8\xb9\xba\xbb\xbc\xbd\xbe\xbf\xc0\xc1\xc2\xc3\xc4\xc5\xc6\xc7\xc8\xc9\xca\xcb\xcc\xcd\xce\xcf\xd0\xd1\xd2\xd3\xd4\xd5\xd6\xd7\xd8\xd9\xda\xdb\xdc\xdd\xde\xdf\xe0\xe1\xe2\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff"
const quotedArgShouldEscapeTable = "\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x01\x01\x01\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x01\x01\x01\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x01\x01\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01"
const quotedPathShouldEscapeTable = "\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x01\x00\x01\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x01\x01\x01\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x01\x01\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01"
const validHeaderFieldByteTable = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x01\x01\x01\x01\x01\x00\x00\x01\x01\x00\x01\x01\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x00\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x01\x00\x01\x00"
const validHeaderValueByteTable = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01"
const validMethodValueByteTable = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x01\x01\x01\x01\x01\x00\x00\x01\x01\x00\x01\x01\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x00\x00\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x00\x01\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

func allocArg(h []argsKV) ([]argsKV, *argsKV) {
	n := len(h)
	if cap(h) > n {
		h = h[:n+1]
	} else {
		h = append(h, argsKV{
			value: []byte{},
		})
	}
	return h, &h[n]
}
func releaseArg(h []argsKV) []argsKV {
	return h[:len(h)-1]
}

// AppendHTTPDate appends HTTP-compliant (RFC1123) representation of date
// to dst and returns the extended dst.
func AppendHTTPDate(dst []byte, date time.Time) []byte {
	dst = date.In(time.UTC).AppendFormat(dst, time.RFC1123)
	copy(dst[len(dst)-3:], strGMT)
	return dst
}

// AppendUint appends n to dst and returns the extended dst.
func AppendUint(dst []byte, n int) []byte {
	if n < 0 {
		// developer sanity-check
		panic("BUG: int must be positive")
	}

	var b [20]byte
	buf := b[:]
	i := len(buf)
	var q int
	for n >= 10 {
		i--
		q = n / 10
		buf[i] = '0' + byte(n-q*10)
		n = q
	}
	i--
	buf[i] = '0' + byte(n)

	dst = append(dst, buf[i:]...)
	return dst
}

// ParseUint parses uint from buf.
func ParseUint(buf []byte) (int, error) {
	v, n, err := parseUintBuf(buf)
	if n != len(buf) {
		return -1, errUnexpectedTrailingChar
	}
	return v, err
}

func parseUintBuf(b []byte) (int, int, error) {
	n := len(b)
	if n == 0 {
		return -1, 0, errEmptyInt
	}
	v := 0
	for i := 0; i < n; i++ {
		c := b[i]
		k := c - '0'
		if k > 9 {
			if i == 0 {
				return -1, i, errUnexpectedFirstChar
			}
			return v, i, nil
		}
		vNew := 10*v + int(k)
		// Test for overflow.
		if vNew < v {
			return -1, i, errTooLongInt
		}
		v = vNew
	}
	return v, n, nil
}

var (
	errEmptyInt               = errors.New("empty integer")
	errUnexpectedFirstChar    = errors.New("unexpected first char found. Expecting 0-9")
	errUnexpectedTrailingChar = errors.New("unexpected trailing char found. Expecting 0-9")
	errTooLongInt             = errors.New("too long int")
)
