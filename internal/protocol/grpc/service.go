package grpc

import (
	"context"
	"fmt"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/gidyon/rupacinema/movie/pkg/api"
	"github.com/gidyon/rupacinema/scheduling/internal/service"
	"github.com/gidyon/rupacinema/scheduling/pkg/api"
	"github.com/gidyon/rupacinema/scheduling/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Creates the service
func createSchedulerServer(
	ctx context.Context, cfg *config.Config,
) (scheduler.ShowSchedulerServer, error) {

	// Remote services
	// MovieService service
	movieServiceConn, err := dialDialMovieServiceService(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to movie service: %v", err)
	}
	// Account service
	accountServiceConn, err := dialAccountService(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to account service: %v", err)
	}

	// Close down all connection when context cancel
	go func() {
		<-ctx.Done()
		movieServiceConn.Close()
		accountServiceConn.Close()
	}()

	return service.NewShowScheduler(
		ctx,
		account.NewAccountAPIClient(accountServiceConn),
		movie.NewMovieAPIClient(movieServiceConn),
	)
}

// creates a connection to the movie service
func dialDialMovieServiceService(
	ctx context.Context, cfg *config.Config,
) (*grpc.ClientConn, error) {

	creds, err := credentials.NewClientTLSFromFile(cfg.MovieAPICertPath, "rupa-movie")
	if err != nil {
		return nil, err
	}

	return grpc.DialContext(
		ctx,
		cfg.MovieAPIAddress+cfg.MovieAPIPort,
		grpc.WithTransportCredentials(creds),
	)
}

// creates a connection to the accounts service
func dialAccountService(
	ctx context.Context, cfg *config.Config,
) (*grpc.ClientConn, error) {

	creds, err := credentials.NewClientTLSFromFile(cfg.AccountServiceCertPath, "rupa-account")
	if err != nil {
		return nil, err
	}

	return grpc.DialContext(
		ctx,
		cfg.AccountServiceAddress+cfg.AccountServicePort,
		grpc.WithTransportCredentials(creds),
	)
}
