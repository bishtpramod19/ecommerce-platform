package server

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	grpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC requests handled.",
		},
		[]string{"service", "method", "code"},
	)

	grpcRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "gRPC request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)
)

// MetricsUnaryInterceptor records count and latency for every unary RPC.
func MetricsUnaryInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req) // run the actual RPC

		// status.Code(nil) returns codes.OK, so success and failure both work
		code := status.Code(err).String()

		grpcRequestsTotal.WithLabelValues(serviceName, info.FullMethod, code).Inc()
		grpcRequestDuration.WithLabelValues(serviceName, info.FullMethod).
			Observe(time.Since(start).Seconds())

		return resp, err
	}
}
