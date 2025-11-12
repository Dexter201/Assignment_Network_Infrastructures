package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metricsHandler is a helper to handle the metrics evolution for us
type MetricsHandler struct {
	requestsTotal  *prometheus.CounterVec
	requestLatency *prometheus.HistogramVec
}

// responseWriterInterceptor is a wrapper for http.ResponseWriter
// to capture the status code.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

// creates a new instance of a MetricsHandler that handles total resuest counts and latencies
func createMetricsHandler() *MetricsHandler {

	metricsHandler := &MetricsHandler{}

	//promauto is a convenience package for Prometheus that automatically registers our metrics.
	metricsHandler.requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of requests to the gateway.",
		},
		[]string{"method", "path", "status", "user"},
	)

	metricsHandler.requestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_latency_seconds",
			Help:    "Latency of requests to the gateway.",
			Buckets: []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 5}, // Buckets in seconds
		},
		[]string{"method", "path", "user"},
	)

	return metricsHandler
}

func createResponseWriterInterceptor(writer http.ResponseWriter) *responseWriterInterceptor {
	return &responseWriterInterceptor{
		ResponseWriter: writer,
		statusCode:     http.StatusOK, // Default to 200 OK
	}
}

func (metricsHandler *MetricsHandler) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, receiver *http.Request) {
		startTime := time.Now()

		//intereptors are wrappers to capture information, in our case we want the statuscode
		interceptor := createResponseWriterInterceptor(writer)

		callNextHandler(next, interceptor, receiver)

		latency := time.Since(startTime).Seconds()
		metricsHandler.saveMetrics(receiver, interceptor.statusCode, latency)
	})
}

func (metricsHandler *MetricsHandler) saveMetrics(receiver *http.Request, statusCode int, latency float64) {
	urlPath := receiver.URL.Path
	method := receiver.Method
	status := strconv.Itoa(statusCode) //integer to ASCII --> we only save strings in the Metrics

	userID := "unknown" //default value
	// here we use our context from auth to pass on internal data
	if id, ok := receiver.Context().Value(userIDKey).(string); ok {
		userID = id
	}

	// Log the metrics saved
	log.Printf("Metrics: user=%s %s %s %d %.4fs", userID, method, urlPath, statusCode, latency)

	//Inc activates the count increase of the count vector
	metricsHandler.requestsTotal.WithLabelValues(method, urlPath, status, userID).Inc()
	//Observe activates the historgam check of the histogram vector
	metricsHandler.requestLatency.WithLabelValues(method, urlPath, userID).Observe(latency)
}

// WriteHeader captures the status code before writing it
func (interceptor *responseWriterInterceptor) WriteHeader(statusCode int) {
	interceptor.statusCode = statusCode
	interceptor.ResponseWriter.WriteHeader(statusCode)
}
