package negroniprometheus

import (
	"net/http"
	"strconv"
	"strings"
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
	routes  []string
}

// NewMiddleware returns a new prometheus Middleware handler.
func NewMiddleware(name string, routes []string, buckets ...float64) *Middleware {
	var m Middleware
	m.routes = routes
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
	start := time.Now()
	res := negroni.NewResponseWriter(rw)
	next(res, r)
	url := "other"
	for _, u := range m.routes {
		if strings.HasPrefix(r.URL.Path, u) {
			url = r.URL.Path
			break
		}
	}
	m.reqs.WithLabelValues(strconv.Itoa(res.Status()), r.Method, url).Inc()
	m.latency.WithLabelValues(strconv.Itoa(res.Status()), r.Method, url).Observe(float64(time.Since(start).Nanoseconds()) / 1000000)
}
