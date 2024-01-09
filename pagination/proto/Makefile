proto: lint-proto generate-proto

generate-proto: ## Regenerate proto files.
	buf generate --template=./buf.gen.yaml --path=pagination

lint-proto:
	buf lint -v --config=buf.yaml --path=pagination