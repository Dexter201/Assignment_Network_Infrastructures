package main

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// we will use HTTP request multiplexers to to route our traffic to the correct endpoints
// the router will multiplex the traffic to the endpoints and create our needed proxies
func createRouter(authHandler *Handler, metricsHandler *MetricsHandler, config *Config) (http.Handler, error) {

	mux := http.NewServeMux()

	userProxy, err := createProxy(config.UserServiceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create user proxy: %w", err)
	}
	postProxy, err := createProxy(config.PostServiceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create post proxy: %w", err)
	}
	feedProxy, err := createProxy(config.FeedServiceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create feed proxy: %w", err)
	}

	healthcheck(mux)

	//------ handle all endpoints ------
	// info: chain start: metrics middleware for everyone

	//prometheus (given by library)
	mux.Handle("/metrics", promhttp.Handler())

	//auth --> we are unauthenticated
	//Chain: Request -> Mux -> metrics.Middleware -> auth.loginHandler
	//does not need striping -> does not pass through proxy
	mux.Handle("/api/auth/register", metricsHandler.metricsMiddleware(http.HandlerFunc(authHandler.register)))
	mux.Handle("/api/auth/login", metricsHandler.metricsMiddleware(http.HandlerFunc(authHandler.login)))

	//authenticated
	//Chain: Request -> Mux -> auth.validationMiddleware -> metrics.Middleware -> proxy.Handler -> (Some Downstream Service)
	//need to strip of "api" for correct proxy handling

	//user Service
	userHandler := authHandler.validationMiddleware(metricsHandler.metricsMiddleware(http.StripPrefix("/api", userProxy)))
	mux.Handle("/api/profile/", userHandler) // Catches /api/profile/me and /api/profile/{userId}
	mux.Handle("/api/friends", userHandler)

	//post Service
	postHandler := authHandler.validationMiddleware(metricsHandler.metricsMiddleware(http.StripPrefix("/api", postProxy)))
	mux.Handle("/api/posts/", postHandler) // Catches /api/posts/me and /api/posts/{userId}

	//feed Service
	feedHandler := authHandler.validationMiddleware(metricsHandler.metricsMiddleware(http.StripPrefix("/api", feedProxy)))
	mux.Handle("/api/feed", feedHandler)

	return mux, nil
}
