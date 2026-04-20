package api

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type requestTrustConfig struct {
	loopbackOnly   bool
	trustedProxies []*net.IPNet
}

func parseTrustedProxyCIDRs(raw string) ([]*net.IPNet, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	items := make([]*net.IPNet, 0)
	for _, candidate := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}

		if !strings.Contains(trimmed, "/") {
			if ip, ok := parseConcreteIP(trimmed); ok {
				bits := 128
				if ip.To4() != nil {
					bits = 32
				}
				trimmed = ip.String() + "/" + strconv.Itoa(bits)
			}
		}

		_, network, err := net.ParseCIDR(trimmed)
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy %q: %w", trimmed, err)
		}
		items = append(items, network)
	}
	return items, nil
}

func (c requestTrustConfig) forwardedHeadersTrusted(r *http.Request) bool {
	if r == nil || c.loopbackOnly || len(c.trustedProxies) == 0 {
		return false
	}

	peerIP, ok := remoteAddrIP(r.RemoteAddr)
	if !ok {
		return false
	}
	for _, network := range c.trustedProxies {
		if network != nil && network.Contains(peerIP) {
			return true
		}
	}
	return false
}

func (c requestTrustConfig) requestScheme(r *http.Request) string {
	if c.forwardedHeadersTrusted(r) {
		if forwarded := strings.TrimSpace(strings.Split(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), ",")[0]); forwarded != "" {
			return strings.ToLower(forwarded)
		}
	}
	return requestSchemeStrict(r)
}

func (c requestTrustConfig) peerIP(r *http.Request) (net.IP, bool) {
	if r == nil {
		return nil, false
	}
	if c.forwardedHeadersTrusted(r) {
		if forwarded, ok := forwardedForIP(r.Header.Get("X-Forwarded-For")); ok {
			return forwarded, true
		}
		if forwarded, ok := parseConcreteIP(r.Header.Get("X-Real-IP")); ok {
			return forwarded, true
		}
	}
	return remoteAddrIP(r.RemoteAddr)
}

func requestSchemeStrict(r *http.Request) string {
	if r != nil && r.TLS != nil {
		return "https"
	}
	return "http"
}

func remoteAddrIP(remoteAddr string) (net.IP, bool) {
	trimmed := strings.TrimSpace(remoteAddr)
	if trimmed == "" {
		return nil, false
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	}
	return parseConcreteIP(trimmed)
}

func forwardedForIP(value string) (net.IP, bool) {
	first := strings.TrimSpace(strings.Split(strings.TrimSpace(value), ",")[0])
	if first == "" {
		return nil, false
	}
	trimmed := strings.Trim(strings.TrimSpace(first), "[]")
	base, hadZone := splitIPZone(trimmed)
	if hadZone {
		return nil, false
	}

	ip, ok := parseConcreteIP(base)
	if !ok {
		return nil, false
	}
	if ip.String() != strings.TrimSpace(base) {
		return nil, false
	}
	return ip, true
}

func parseConcreteIP(value string) (net.IP, bool) {
	trimmed := strings.Trim(strings.TrimSpace(value), "[]")
	if trimmed == "" {
		return nil, false
	}
	base, hadZone := splitIPZone(trimmed)
	ip := net.ParseIP(base)
	if ip == nil {
		return nil, false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		if hadZone {
			return nil, false
		}
		return ipv4, true
	}
	return ip, true
}

func splitIPZone(value string) (string, bool) {
	base, _, found := strings.Cut(strings.TrimSpace(value), "%")
	return base, found
}

func sameOriginRequest(r *http.Request, origin string) bool {
	return sameOriginHost(requestSchemeStrict(r), requestHost(r), origin)
}

func sameOriginHost(requestScheme, requestHost, origin string) bool {
	originKey, ok := normalizedOriginKey(origin)
	if !ok {
		return false
	}
	requestKey, ok := normalizedSchemeHostKey(requestScheme, requestHost)
	if !ok {
		return false
	}
	return originKey == requestKey
}

func normalizedOriginKey(origin string) (string, bool) {
	parsedOrigin, err := url.Parse(strings.TrimSpace(origin))
	if err != nil {
		return "", false
	}
	return normalizedSchemeHostKey(parsedOrigin.Scheme, parsedOrigin.Host)
}

func normalizedSchemeHostKey(rawScheme, rawHost string) (string, bool) {
	scheme := strings.ToLower(strings.TrimSpace(rawScheme))
	if scheme == "" {
		return "", false
	}

	hostPort, ok := normalizedHostPort(rawHost, scheme)
	if !ok {
		return "", false
	}
	return scheme + "://" + hostPort, true
}

func normalizedHostPort(rawHost, scheme string) (string, bool) {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	rawHost = strings.TrimSpace(rawHost)
	if rawHost == "" {
		return "", false
	}

	parsed := &url.URL{Scheme: scheme, Host: rawHost}
	host := strings.TrimSpace(parsed.Hostname())
	port := strings.TrimSpace(parsed.Port())
	if host == "" {
		host = strings.Trim(strings.TrimSpace(rawHost), "[]")
	}
	if host == "" {
		return "", false
	}
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}

	if ip, ok := parseConcreteIP(host); ok {
		host = ip.String()
	} else {
		host = strings.ToLower(host)
	}
	if port == "" {
		return host, true
	}
	return net.JoinHostPort(host, port), true
}

func requestHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Host)
}
