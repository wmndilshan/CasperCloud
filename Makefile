.PHONY: swagger

# Generates OpenAPI (Swagger 2) YAML/JSON under ./api and copies swagger.yaml to openapi.yaml.
swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -g ./cmd/api/main.go -o ./api --parseDependency --parseInternal
	cp -f ./api/swagger.yaml ./api/openapi.yaml
