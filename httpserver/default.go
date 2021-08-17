package httpserver

import (
	"context"
	"github.com/purposeinplay/go-commons/http/router"
	"github.com/purposeinplay/go-commons/logs"
)

func NewDefaultServer(ctx context.Context, r router.Router) *Server {
	logger := logs.NewLogger()

	//apiRouter := router.NewDefaultRouter(logger)
	//rootRouter := router.New()

	// we are mounting all APIs under /api path
	//rootRouter.Mount("/api", r)

	return New(
		logger,
		r,
		WithBaseContext(ctx, true),
	)
}
