package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/lxc/incus/v7/internal/server/auth"
	"github.com/lxc/incus/v7/internal/server/response"
	"github.com/lxc/incus/v7/shared/logger"
)

// osProxyTransport is shared by all requests to IncusOS so that its connections are reused.
var osProxyTransport = &http.Transport{
	DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer

		return d.DialContext(ctx, "unix", "/run/incus-os/unix.socket")
	},
	// Below the idle timeout of IncusOS, so idle connections are closed by us first.
	IdleConnTimeout: 5 * time.Second,
}

// osProxyErrorHandler reports a failed request to IncusOS, unless the client went away.
func osProxyErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if r.Context().Err() != nil {
		return
	}

	logger.Warn("Failed proxying request to IncusOS", logger.Ctx{"url": r.URL.String(), "err": err})
	w.WriteHeader(http.StatusBadGateway)
}

var apiOS = APIEndpoint{
	Path:   "{name...}",
	Patch:  APIEndpointAction{Handler: apiOSProxy, AccessHandler: allowPermission(auth.ObjectTypeServer, auth.EntitlementCanEdit)},
	Put:    APIEndpointAction{Handler: apiOSProxy, AccessHandler: allowPermission(auth.ObjectTypeServer, auth.EntitlementCanEdit)},
	Get:    APIEndpointAction{Handler: apiOSProxy, AccessHandler: allowPermission(auth.ObjectTypeServer, auth.EntitlementCanEdit)},
	Post:   APIEndpointAction{Handler: apiOSProxy, AccessHandler: allowPermission(auth.ObjectTypeServer, auth.EntitlementCanEdit)},
	Delete: APIEndpointAction{Handler: apiOSProxy, AccessHandler: allowPermission(auth.ObjectTypeServer, auth.EntitlementCanEdit)},
	Head:   APIEndpointAction{Handler: apiOSProxy, AccessHandler: allowPermission(auth.ObjectTypeServer, auth.EntitlementCanEdit)},
}

func apiOSProxy(d *Daemon, r *http.Request) response.Response {
	s := d.State()

	// If a target was specified, forward the request to the relevant node.
	resp := forwardedResponseIfTargetIsRemote(s, r)
	if resp != nil {
		return resp
	}

	// Check if this is an IncusOS system.
	if s.OS.IncusOS == nil {
		return response.BadRequest(errors.New("System isn't running IncusOS"))
	}

	// Prepare the proxy.
	proxy := &httputil.ReverseProxy{
		Transport: osProxyTransport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.Out.URL.Scheme = "http"
			r.Out.URL.Host = "incus-os"
		},
		ErrorHandler: osProxyErrorHandler,
	}

	// Allow IncusOS to adjust the returned paths to the prefix used by the proxy.
	r.Header.Add("X-IncusOS-Proxy", "/os")

	// Handle the request.
	return response.ManualResponse(func(w http.ResponseWriter) error {
		http.StripPrefix("/os", proxy).ServeHTTP(w, r)

		return nil
	})
}
