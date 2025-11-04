package smartbear

// nolint: revive
//go:generate go tool oapi-codegen -generate=types,skip-prune,spec -o problemdetails.gen.go -package=smartbear https://api.swaggerhub.com/domains/smartbear-public/ProblemDetails/1.0.0
