package http

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/newacorn/fasthttp"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
	"strings"
	"time"
)

type ProxyDialerFunc func(network, addr string) (net.Conn, error)

func (pd ProxyDialerFunc) Dial(network, addr string) (net.Conn, error) {
	return pd(network, addr)
}

type ProxyTCPDialer struct {
	fasthttp.TCPDialer
	timeout time.Duration
}

func init() {
	proxy.RegisterDialerType("http", func(u *url.URL, dialer proxy.Dialer) (proxy.Dialer, error) {
		proxyAddr := u.String()
		var auth string
		if strings.Contains(proxyAddr, "@") {
			index := strings.LastIndex(proxyAddr, "@")
			auth = base64.StdEncoding.EncodeToString([]byte(proxyAddr[:index]))
			proxyAddr = proxyAddr[index+1:]
		}
		f := func(network, addr string) (conn net.Conn, err error) {
			conn, err = dialer.Dial(network, addr)
			if err != nil {
				return
			}
			req := "CONNECT " + addr + " HTTP/1.1\r\nHost: " + addr + "\r\n"
			if auth != "" {
				req += "Proxy-Authorization: Basic " + auth + "\r\n"
			}
			req += "\r\n"
			_, err = conn.Write([]byte(req))
			if err != nil {
				_ = conn.Close()
				return
			}
			res := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseResponse(res)
			res.SkipBody = true
			if err = res.Read(bufio.NewReader(conn)); err != nil {
				_ = conn.Close()
				return
			}
			if res.Header.StatusCode() != 200 {
				_ = conn.Close()
				err = fmt.Errorf("could not connect to proxyAddr: %s status code: %d", proxyAddr, res.Header.StatusCode())
				return
			}
			return
		}
		return (ProxyDialerFunc)(f), nil
	})
}
func (c *ProxyTCPDialer) Dial(network, addr string) (conn net.Conn, err error) {
	if network == "tcp4" {
		if c.timeout > 0 {
			return c.TCPDialer.DialTimeout(addr, c.timeout)
		}
		return c.TCPDialer.Dial(addr)
	}
	if network == "tcp" {
		if c.timeout > 0 {
			return c.TCPDialer.DialDualStackTimeout(addr, c.timeout)
		}
		return c.TCPDialer.DialDualStack(addr)
	}
	err = errors.New("don't know how to dial network:" + network)
	return
}

func (c *ProxyTCPDialer) GetDialFunc(proxyAddr string) (dialFunc fasthttp.DialFunc, err error) {
	u, err := url.Parse(proxyAddr)
	if err != nil {
		return
	}
	dialer, err := proxy.FromURL(u, c)
	if err != nil {
		return
	}
	dialFunc = func(addr string) (net.Conn, error) {
		var network string
		if strings.HasPrefix(proxyAddr, "[") {
			network = "tcp"
		} else {
			network = "tcp4"
		}
		return dialer.Dial(network, addr)
	}
	return
}
