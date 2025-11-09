package gateway

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// createProxy creates a reverse proxy that forwards requests to the given target URL.
// It adjusts the Host and X-Forwarded-Host headers to avoid 421 errors and sets a custom
// error handler.
func createProxy(targetURL string) (*httputil.ReverseProxy, error) {

	target, err := url.Parse(targetURL)
	if err != nil {
		log.Printf("Failed to parse target URL: %v", err)
		return nil, err
	}

	//forwards to the URL but keeps original Host Header
	// Problem --> Host=lcoalhost:443 but we need x-load-balancer:8080 or we have a 421 error (misdirection)
	proxy := httputil.NewSingleHostReverseProxy(target)

	// The Director is a function that runs just before the request is forwarded.
	//This is where we can change the header of the requests so that the proxy adds the correct Host URL
	originalDirector := proxy.Director
	proxy.Director = func(request *http.Request) {
		originalDirector(request)
		request.Host = target.Host

		// 3b. Inject the 'X-Forwarded-Host' header.
		request.Header.Set("X-Forwarded-Host", request.Host)
		// We don't set "X-User-ID" here since authMiddleware already set it
		// The proxy shall just forward it automatically for ease
	}

	proxy.ErrorHandler = func(writer http.ResponseWriter, receiver *http.Request, err error) {
		log.Printf("Proxy error for %s: %v", receiver.URL, err)
		http.Error(writer, "Bad Gateway", http.StatusBadGateway)
	}

	return proxy, nil
}
