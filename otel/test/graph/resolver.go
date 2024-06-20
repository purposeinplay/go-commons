package graph

import (
	"context"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/purposeinplay/go-commons/otel/test/graph/generated"
	"github.com/purposeinplay/go-commons/otel/test/graph/model"
	"github.com/ravilushqa/otelgqlgen"
)

//go:generate go run github.com/99designs/gqlgen generate

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver is the root resolver for the GraphQL API.
type Resolver struct {
	GetUserIDFunc func(ctx context.Context, id string) (*model.User, error)
}

// NewServer creates a new GraphQL server.
func NewServer(
	getUserIDFunc func(ctx context.Context, id string) (*model.User, error),
) *handler.Server {
	gqlServer := handler.NewDefaultServer(generated.NewExecutableSchema(
		generated.Config{
			Resolvers: &Resolver{
				GetUserIDFunc: getUserIDFunc,
			},
		}))

	gqlServer.Use(otelgqlgen.Middleware())

	return gqlServer
}
