//go:build unix || (js && wasm) || wasip1

package goutils

import (
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
)

// NoBlockingRead net.ErrClosed
// syscall.EAGAIN syscall.ERROR_IO_PENDING
func NoBlockingRead(c *net.TCPConn, p []byte) (n int, err error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return
	}
	err1 := raw.Read(func(fd uintptr) (done bool) {
		n, err = syscall.Read(int(fd), p[:])
		if n == 0 && err == nil {
			err = io.EOF
		}
		return true
	})
	if err == nil {
		err = err1
	}
	return
}

func SplitHostAndPort(h string) (host string, port string, err error) {
	if len(h) < 3 {
		err = fmt.Errorf("invalid hostport: %q", h)
		return
	}
	c := strings.Count(h, ":")
	if c == 0 {
		host = h
		return
	}
	lastSemiIdx := strings.LastIndex(h, ":")
	if c == 1 {
		host, port = h[:lastSemiIdx], h[lastSemiIdx+1:]
		return
	}
	if lastSemiIdx > 0 {
		if h[lastSemiIdx-1] == ']' {
			host = h[1 : lastSemiIdx-1]
			port = h[lastSemiIdx+1:]
			return
		}
	}
	if h[0] == '[' && h[len(h)-1] == ']' {
		host = h[1 : len(h)-1]
		return
	}
	host, port, err = net.SplitHostPort(h)
	return
}

// SplitHostAndPortStd splitHostPort separates host and port. If the port is not valid, it returns
// the entire input as host, and it doesn't check the validity of the host.
// Unlike net.SplitHostPort, but per RFC 3986, it requires ports to be numeric.
func SplitHostAndPortStd(hostPort string) (host, port string) {
	host = hostPort

	colon := strings.LastIndexByte(host, ':')
	if colon != -1 && validOptionalPort(host[colon:]) {
		host, port = host[:colon], host[colon+1:]
	}

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	return
}

func validOptionalPort(port string) bool {
	if port == "" {
		return true
	}
	if port[0] != ':' {
		return false
	}
	for _, b := range port[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

// AddMissingPort adds a port to a host if it is missing.
// A literal IPv6 address in hostport must be enclosed in square
// brackets, as in "[::1]:80", "[::1%lo0]:80".
func AddMissingPort(addr string, isTLS bool) string {
	addrLen := len(addr)
	if addrLen == 0 {
		return addr
	}
	isIP6 := addr[0] == '['
	if isIP6 {
		// if the IPv6 has opening bracket but closing bracket is the last char then it doesn't have a port
		isIP6WithoutPort := addr[addrLen-1] == ']'
		if !isIP6WithoutPort {
			return addr
		}
	} else { // IPv4
		lastColumnPos := strings.LastIndexByte(addr, ':')
		if lastColumnPos > 0 {
			fistColumnPos := strings.IndexByte(addr[:lastColumnPos], ':')
			if fistColumnPos > 0 {
				addr = "[" + addr + "]"
			} else {
				return addr
			}
		}
	}
	port := ":80"
	if isTLS {
		port = ":443"
	}
	return addr + port
}

func ExtractHostname(addr string) (hostname string) {
	c := strings.Count(addr, ":")
	if c == 0 {
		return
	}
	lastSemiIdx := strings.LastIndex(addr, ":")
	if c == 1 {
		hostname = addr[:lastSemiIdx]
		return
	}
	if lastSemiIdx > 0 {
		if addr[lastSemiIdx-1] == ']' {
			hostname = addr[:lastSemiIdx]
		}
	}
	if len(addr) == 0 {
		return
	}
	if addr[0] == '[' && addr[len(addr)-1] == ']' {
		hostname = addr[1 : len(addr)-1]
		return
	}
	return
}

func HasPort(s string) bool { return strings.LastIndex(s, ":") > strings.LastIndex(s, "]") }
