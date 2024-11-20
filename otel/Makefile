.PHONY: test

test:
	go test -race ./...
lint:
	golangci-lint run --fix -c=../.golangci.yml
start-collector:
	docker compose -f ./test/docker-compose.yaml up
validate-config:
	docker run --volume=./test/collector-config.yaml:/etc/otel/config.yaml otel/opentelemetry-collector-contrib validate --config=/etc/otel/config.yaml
create-sample-traces:
	telemetrygen traces --otlp-insecure --traces 3