package oauth

import (
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// redirectURIMatches permits only the port variation required for native loopback clients by RFC 8252.
func redirectURIMatches(registered, requested string) bool {
	registration, err := url.Parse(registered)
	if err != nil {
		return false
	}
	if registration.Scheme != "http" {
		return registered == requested
	}
	address, err := netip.ParseAddr(registration.Hostname())
	if err != nil || !address.IsLoopback() {
		return registered == requested
	}

	redirect, err := url.Parse(requested)
	if err != nil {
		return false
	}
	for _, uri := range []*url.URL{registration, redirect} {
		if uri.User != nil || uri.Fragment != "" || uri.RawFragment != "" || uri.Opaque != "" || strings.HasSuffix(uri.Host, ":") {
			return false
		}
		if port := uri.Port(); port != "" {
			number, err := strconv.Atoi(port)
			if err != nil || number < 1 || number > 65535 {
				return false
			}
		}
	}
	if strings.Contains(registered, "#") || strings.Contains(requested, "#") {
		return false
	}
	return registration.Scheme == redirect.Scheme &&
		registration.Hostname() == redirect.Hostname() &&
		registration.EscapedPath() == redirect.EscapedPath() &&
		registration.RawQuery == redirect.RawQuery &&
		registration.ForceQuery == redirect.ForceQuery
}
