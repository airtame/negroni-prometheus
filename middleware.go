package negroniprometheus

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/negroni"
)

var (
	dflBuckets = []float64{300, 1200, 5000}
)

const (
	reqsName    = "negroni_requests_total"
	latencyName = "negroni_request_duration_milliseconds"
)

// Middleware is a handler that exposes prometheus metrics for the number of requests,
// the latency and the response size, partitioned by status code, method and HTTP path.
type Middleware struct {
	reqs    *prometheus.CounterVec
	latency *prometheus.HistogramVec
}

type matchKey struct{}

// MatchedRoutePathKey is the request context key under which the handler path
// match is stored.
var MatchedRoutePathKey = matchKey{}

// NewMiddleware returns a new prometheus Middleware handler.
func NewMiddleware(name string, buckets ...float64) *Middleware {
	var m Middleware
	m.reqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        reqsName,
			Help:        "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
			ConstLabels: prometheus.Labels{"service": name},
		},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(m.reqs)

	if len(buckets) == 0 {
		buckets = dflBuckets
	}
	m.latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        latencyName,
		Help:        "How long it took to process the request, partitioned by status code, method and HTTP path.",
		ConstLabels: prometheus.Labels{"service": name},
		Buckets:     buckets,
	},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(m.latency)
	return &m
}

func (m *Middleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	var path string

	p, ok := r.Context().Value(MatchedRoutePathKey).(string)
	if ok {
		path = p
	} else {
		path = r.URL.Path
	}

	start := time.Now()
	next(rw, r)
	res := negroni.NewResponseWriter(rw)
	m.reqs.WithLabelValues(http.StatusText(res.Status()), r.Method, path).Inc()
	m.latency.WithLabelValues(http.StatusText(res.Status()), r.Method, path).Observe(float64(time.Since(start).Nanoseconds()) / 1000000)
}
