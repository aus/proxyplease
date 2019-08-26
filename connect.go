package proxyplease

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func dialAndNegotiateHTTP(p Proxy, addr string, baseDial func() (net.Conn, error)) (net.Conn, error) {
	// establish TCP with proxy. baseDial will negoiate TLS if needed.
	conn, err := baseDial()
	if err != nil {
		debugf("connect> Could not call dial context with proxy: %s", err)
		return conn, err
	}

	// build and write first CONNECT request
	connect := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: *p.Headers,
	}
	if err := connect.Write(conn); err != nil {
		debugf("connect> CONNECT to proxy failed: %s", err)
		return conn, err
	}

	// read first response
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connect)
	if err != nil {
		debugf("connect> Could not read response from proxy: %s", err)
		return conn, err
	}

	// if StatusOK, no auth is required and proxy is established
	if resp.StatusCode == http.StatusOK {
		debugf("connect> Proxy successfully established. No authentication was required.")
		return conn, nil
	}

	// if authentication is required
	if resp.StatusCode == http.StatusProxyAuthRequired {
		debugf("connect> Proxy authentication is required. Attempting to select a authentication scheme.")

		// read authentication scheme options
		schemes := resp.Header["Proxy-Authenticate"]
		for _, s := range schemes {
			// only test for first word in scheme
			trimmed := strings.Split(s, " ")[0]
			switch trimmed {
			case "NTLM":
				conn, err = dialNTLM(p, addr, baseDial)
				if err != nil {
					debugf("connect> NTLM authentication failed. Trying next available scheme.")
					continue
				}
				return conn, err
			case "Basic":
				conn, err = dialBasic(p, addr, baseDial)
				if err != nil {
					debugf("connect> Basic authentication failed. Trying next available scheme.")
					continue
				}
				return conn, err

			case "Negotiate":
				conn, err = dialNegotiate(p, addr, baseDial)
				if err != nil {
					debugf("connect> Negotiate authentication failed. Trying next available scheme.")
					continue
				}
				return conn, err

			case "Kerberos":
				debugf("connect> Kerberos not implemented yet. Trying next available scheme.")
				continue

			case "Digest":
				debugf("connect> Digest not implemented yet. Trying next available scheme.")
				continue

			default:
				debugf("connect> Unsupported proxy authentication scheme: '%s'. Trying next available scheme.", trimmed)
				continue
			}
		}

		debugf("connect> No proxy authentication completed successfully")
		return conn, err
	}

	debugf("connect> Unhandled HTTP status, got: %d", resp.StatusCode)
	return conn, errors.New(http.StatusText(resp.StatusCode))
}
