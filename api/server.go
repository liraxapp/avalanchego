// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/handlers"

	"github.com/rs/cors"

	"github.com/liraxapp/avalanchego/api/auth"
	"github.com/liraxapp/avalanchego/snow"
	"github.com/liraxapp/avalanchego/snow/engine/common"
	"github.com/liraxapp/avalanchego/utils/logging"
)

const (
	baseURL               = "/ext"
	serverShutdownTimeout = 10 * time.Second
)

var (
	errUnknownLockOption = errors.New("invalid lock options")
)

// Server maintains the HTTP router
type Server struct {
	// log this server writes to
	log logging.Logger
	// generates new logs for chains to write to
	factory logging.Factory
	// Maps endpoints to handlers
	router *router
	// Listens for HTTP traffic on this address
	listenAddress string
	// Handles authorization. Must be non-nil after initialization, even if
	// token authorization is off.
	auth *auth.Auth

	// http server
	srv *http.Server
}

// Initialize creates the API server at the provided host and port
func (s *Server) Initialize(
	log logging.Logger,
	factory logging.Factory,
	host string,
	port uint16,
	authEnabled bool,
	authPassword string,
) error {
	s.log = log
	s.factory = factory
	s.listenAddress = fmt.Sprintf("%s:%d", host, port)
	s.router = newRouter()
	s.auth = &auth.Auth{Enabled: authEnabled}
	if err := s.auth.Password.Set(authPassword); err != nil {
		return err
	}
	if !authEnabled {
		return nil
	}

	// only create auth service if token authorization is required
	s.log.Info("API authorization is enabled. Auth tokens must be passed in the header of API requests, except requests to the auth service.")
	authService := auth.NewService(s.log, s.auth)
	return s.AddRoute(authService, &sync.RWMutex{}, auth.Endpoint, "", s.log)

}

// Dispatch starts the API server
func (s *Server) Dispatch() error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return err
	}
	s.log.Info("HTTP API server listening on %q", s.listenAddress)
	handler := cors.Default().Handler(s.router)
	handler = s.auth.WrapHandler(handler)
	s.srv = &http.Server{Handler: handler}
	return s.srv.Serve(listener)
}

// DispatchTLS starts the API server with the provided TLS certificate
func (s *Server) DispatchTLS(certFile, keyFile string) error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return err
	}
	s.log.Info("HTTPS API server listening on %q", s.listenAddress)
	handler := cors.Default().Handler(s.router)
	handler = s.auth.WrapHandler(handler)
	return http.ServeTLS(listener, handler, certFile, keyFile)
}

// RegisterChain registers the API endpoints associated with this chain That is,
// add <route, handler> pairs to server so that http calls can be made to the vm
func (s *Server) RegisterChain(chainName string, ctx *snow.Context, vmIntf interface{}) {
	vm, ok := vmIntf.(common.VM)
	if !ok {
		return
	}

	httpLogger, err := s.factory.MakeChain(chainName, "http")
	if err != nil {
		s.log.Error("Failed to create new http logger: %s", err)
		return
	}

	s.log.Verbo("About to add API endpoints for chain with ID %s", ctx.ChainID)
	// all subroutes to a chain begin with "bc/<the chain's ID>"
	defaultEndpoint := "bc/" + ctx.ChainID.String()

	// Register each endpoint
	for extension, service := range vm.CreateHandlers() {
		// Validate that the route being added is valid
		// e.g. "/foo" and "" are ok but "\n" is not
		_, err := url.ParseRequestURI(extension)
		if extension != "" && err != nil {
			s.log.Error("could not add route to chain's API handler because route is malformed: %s", err)
			continue
		}
		if err := s.AddChainRoute(service, ctx, defaultEndpoint, extension, httpLogger); err != nil {
			s.log.Error("error adding route: %s", err)
		}
	}
}

// AddChainRoute registers a route to a chain's handler
func (s *Server) AddChainRoute(handler *common.HTTPHandler, ctx *snow.Context, base, endpoint string, loggingWriter io.Writer) error {
	url := fmt.Sprintf("%s/%s", baseURL, base)
	s.log.Info("adding route %s%s", url, endpoint)
	// Apply logging middleware
	h := handlers.CombinedLoggingHandler(loggingWriter, handler.Handler)
	// Apply middleware to grab/release chain's lock before/after calling API method
	h, err := lockMiddleware(h, handler.LockOptions, &ctx.Lock)
	if err != nil {
		return err
	}
	// Apply middleware to reject calls to the handler before the chain finishes bootstrapping
	h = rejectMiddleware(h, ctx)
	return s.router.AddRouter(url, endpoint, h)
}

// AddRoute registers a route to a handler.
func (s *Server) AddRoute(handler *common.HTTPHandler, lock *sync.RWMutex, base, endpoint string, loggingWriter io.Writer) error {
	url := fmt.Sprintf("%s/%s", baseURL, base)
	s.log.Info("adding route %s%s", url, endpoint)
	// Apply logging middleware
	h := handlers.CombinedLoggingHandler(loggingWriter, handler.Handler)
	// Apply middleware to grab/release chain's lock before/after calling API method
	h, err := lockMiddleware(h, handler.LockOptions, lock)
	if err != nil {
		return err
	}
	return s.router.AddRouter(url, endpoint, h)
}

// Wraps a handler by grabbing and releasing a lock before calling the handler.
func lockMiddleware(handler http.Handler, lockOption common.LockOption, lock *sync.RWMutex) (http.Handler, error) {
	switch lockOption {
	case common.WriteLock:
		return middlewareHandler{
			before:  lock.Lock,
			after:   lock.Unlock,
			handler: handler,
		}, nil
	case common.ReadLock:
		return middlewareHandler{
			before:  lock.RLock,
			after:   lock.RUnlock,
			handler: handler,
		}, nil
	case common.NoLock:
		return handler, nil
	default:
		return nil, errUnknownLockOption
	}
}

// Reject middleware wraps a handler. If the chain that the context describes is
// not done bootstrapping, writes back an error.
func rejectMiddleware(handler http.Handler, ctx *snow.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // If chain isn't done bootstrapping, ignore API calls
		if !ctx.IsBootstrapped() {
			w.WriteHeader(http.StatusServiceUnavailable)
			// Doesn't matter if there's an error while writing. They'll get the StatusServiceUnavailable code.
			_, _ = w.Write([]byte("API call rejected because chain is not done bootstrapping"))
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

// AddAliases registers aliases to the server
func (s *Server) AddAliases(endpoint string, aliases ...string) error {
	url := fmt.Sprintf("%s/%s", baseURL, endpoint)
	endpoints := make([]string, len(aliases))
	for i, alias := range aliases {
		endpoints[i] = fmt.Sprintf("%s/%s", baseURL, alias)
	}
	return s.router.AddAlias(url, endpoints...)
}

// AddAliasesWithReadLock registers aliases to the server assuming the http read
// lock is currently held.
func (s *Server) AddAliasesWithReadLock(endpoint string, aliases ...string) error {
	// This is safe, as the read lock doesn't actually need to be held once the
	// http handler is called. However, it is unlocked later, so this function
	// must end with the lock held.
	s.router.lock.RUnlock()
	defer s.router.lock.RLock()

	return s.AddAliases(endpoint, aliases...)
}

// Call ...
func (s *Server) Call(
	writer http.ResponseWriter,
	method,
	base,
	endpoint string,
	body io.Reader,
	headers map[string]string,
) error {
	url := fmt.Sprintf("%s/vm/%s", baseURL, base)

	handler, err := s.router.GetHandler(url, endpoint)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", "*", body)
	if err != nil {
		return err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	handler.ServeHTTP(writer, req)

	return nil
}

// Shutdown this server
func (s *Server) Shutdown() error {
	if s.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()
	return s.srv.Shutdown(ctx)
}
