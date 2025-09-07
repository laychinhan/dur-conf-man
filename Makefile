.PHONY: format
format:
	bin/format.sh

.PHONY: tidy
tidy:
	GO111MODULE=on go mod tidy

.PHONY: lint
lint: lint.cleancache
	golangci-lint run ./...

.PHONY: check.import
check.import:
	bin/check-import.sh

.PHONY: lint.cleancache
lint.cleancache:
	golangci-lint cache clean

.PHONY: pretty
pretty: tidy format lint

.PHONY: mod.download
mod.download:
	GO111MODULE=on go mod download

.PHONY: vendor
vendor:
	GO111MODULE=on go mod vendor

.PHONY: api-docs
api-docs:
	swag init -g cmd/server/main.go --parseDependency --parseInternal
