.PHONY: gen-proto
gen-proto: gen

.PHONY: gen
gen:
	protoc \
		--proto_path=internal/grpc/protoc \
		--go_out=internal/grpc/protoc \
		--go_opt=paths=source_relative \
		internal/grpc/protoc/model/*.proto

	protoc \
		--proto_path=internal/grpc/protoc \
		--proto_path=internal/grpc/protoc/model \
		--go_out=internal/grpc/protoc \
		--go_opt=paths=source_relative \
		--go-grpc_out=internal/grpc/protoc \
		--go-grpc_opt=paths=source_relative \
		internal/grpc/protoc/*.proto

.PHONY: build
build: build-server-client

.PHONY: build-server-agent
build-server-client: gen-proto
	go build -o cmd/agent/agent ./cmd/agent
	go build -o cmd/server/server ./cmd/server

.PHONY: clean-gen-proto
clean-gen-proto:
	find internal/protos -type f ! -name "*.proto" -delete

.PHONY: test-coverpkg
test-coverpkg:
	@INCLUDE_PACKAGES=$$(go list ./... | grep -v -E '/mocks|/protoc') && \
	go test -coverpkg=$$(echo $$INCLUDE_PACKAGES | tr ' ' ',') -coverprofile=coverage_raw.out $$INCLUDE_PACKAGES && \
	grep -v -E "go-musthave-metrics/cmd/staticlint/main.go|go-musthave-metrics/cmd/server/main.go" coverage_raw.out > coverage.out && \
	rm coverage_raw.out



.PHONY: gen-mocks
gen-mocks:

