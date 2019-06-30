package grpc

import (
	"context"
	"fmt"
	"github.com/gidyon/rupacinema/scheduling/internal/protocol"
	"github.com/gidyon/rupacinema/scheduling/pkg/config"
	"github.com/grpc-ecosystem/go-grpc-middleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/gidyon/rupacinema/scheduling/internal/protocol/grpc/middleware"
	"github.com/gidyon/rupacinema/scheduling/pkg/api"
	"github.com/gidyon/rupacinema/scheduling/pkg/logger"
)

// CreateGRPCServer ...
func CreateGRPCServer(ctx context.Context, cfg *config.Config) (*grpc.Server, error) {

	tlsConfig, err := protocol.GRPCServerTLS()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(tlsConfig)

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
	}

	err = logger.Init(cfg.LogLevel, cfg.LogTimeFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %v", err)
	}

	// add logging middleware
	unaryLoggerInterceptors, streamLoggerInterceptors := middleware.AddLogging(logger.Log)

	// add recovery from panic middleware
	unaryRecoveryInterceptors, streamRecoveryInterceptors := middleware.AddRecovery()

	opts = append(opts,
		grpc_middleware.WithUnaryServerChain(
			chainUnaryInterceptors(
				unaryLoggerInterceptors,
				unaryRecoveryInterceptors,
			)...,
		),
		grpc_middleware.WithStreamServerChain(
			chainStreamInterceptors(
				streamLoggerInterceptors,
				streamRecoveryInterceptors,
			)...,
		),
	)

	s := grpc.NewServer(opts...)

	schedulingService, err := createSchedulerServer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	scheduler.RegisterShowSchedulerServer(s, schedulingService)

	// Register reflection service on gRPC server.
	reflection.Register(s)

	return s, nil
}

type grpcUnaryInterceptorsSlice []grpc.UnaryServerInterceptor

func chainUnaryInterceptors(
	unaryInterceptorsSlice ...grpcUnaryInterceptorsSlice,
) []grpc.UnaryServerInterceptor {
	unaryInterceptors := make([]grpc.UnaryServerInterceptor, 0, len(unaryInterceptorsSlice))

	for _, unaryInterceptorSlice := range unaryInterceptorsSlice {
		for _, unaryInterceptor := range unaryInterceptorSlice {
			unaryInterceptors = append(unaryInterceptors, unaryInterceptor)
		}
	}

	return unaryInterceptors
}

type grpcStreamInterceptorsSlice []grpc.StreamServerInterceptor

func chainStreamInterceptors(
	streamInterceptorsSlice ...grpcStreamInterceptorsSlice,
) []grpc.StreamServerInterceptor {
	streamInterceptors := make([]grpc.StreamServerInterceptor, 0, len(streamInterceptorsSlice))

	for _, streamInterceptorSlice := range streamInterceptorsSlice {
		for _, streamInterceptor := range streamInterceptorSlice {
			streamInterceptors = append(streamInterceptors, streamInterceptor)
		}
	}

	return streamInterceptors
}
