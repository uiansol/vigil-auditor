.PHONY: all generate proto lint test build modernize compose-up compose-down

# Default verification pipeline for Slice 0+
all: generate test build

# Compile SQL queries into type-safe Go code using sqlc
generate:
	@echo "==> Generating type-safe Go code from SQL..."
	@sqlc generate

# Regenerate gRPC stubs (requires protoc + plugins; see README)
proto:
	@echo "==> Generating protobuf stubs..."
	@mkdir -p pkg/auditorpb services/ai/app/pb
	@PATH="$(CURDIR)/tools/bin:$$(go env GOPATH)/bin:$$PATH" protoc -I api/proto \
		--go_out=pkg/auditorpb --go_opt=module=github.com/uiansol/vigil-auditor/pkg/auditorpb \
		--go-grpc_out=pkg/auditorpb --go-grpc_opt=module=github.com/uiansol/vigil-auditor/pkg/auditorpb \
		api/proto/auditor/v1/auditor.proto
	@if [ -d pkg/auditorpb/auditor/v1 ]; then \
		mv pkg/auditorpb/auditor/v1/*.go pkg/auditorpb/ && rm -rf pkg/auditorpb/auditor; \
	fi
	@echo "==> Python stubs: regenerate via Docker (host may lack python3-venv)"
	@docker run --rm --user "$$(id -u):$$(id -g)" -v "$(CURDIR):/work" -w /work -e HOME=/tmp python:3.12-slim bash -c '\
		pip install -q --user grpcio-tools==1.71.0 protobuf==5.29.3 && \
		python -m grpc_tools.protoc -I api/proto \
			--python_out=services/ai/app/pb \
			--grpc_python_out=services/ai/app/pb \
			--pyi_out=services/ai/app/pb \
			api/proto/auditor/v1/auditor.proto && \
		if [ -d services/ai/app/pb/auditor/v1 ]; then \
			mv services/ai/app/pb/auditor/v1/* services/ai/app/pb/ && rm -rf services/ai/app/pb/auditor; \
		fi'
	@python3 -c "from pathlib import Path; p=Path('services/ai/app/pb/auditor_pb2_grpc.py'); t=p.read_text();\
import re; t=re.sub(r'^import auditor_pb2 as .+', 'from app.pb import auditor_pb2 as auditor_dot_v1_dot_auditor__pb2', t, flags=re.M);\
t=re.sub(r'^from auditor\.v1 import auditor_pb2 as .+', 'from app.pb import auditor_pb2 as auditor_dot_v1_dot_auditor__pb2', t, flags=re.M);\
t=re.sub(r'^from app\.pb from app\.pb import', 'from app.pb import', t, flags=re.M); p.write_text(t)"
	@touch services/ai/app/pb/__init__.py

# Use Go modernizers only on our packages
modernize:
	@echo "==> Running Go modernizers..."
	@go fix ./cmd/... ./internal/... ./pkg/...

# Run advanced static analysis and formatting
lint:
	@echo "==> Formatting imports..."
	@goimports -w ./cmd ./internal ./pkg
	@echo "==> Running golangci-lint..."
	@golangci-lint run ./cmd/... ./internal/... ./pkg/...

# Run the test suite (limit to Go packages; avoid node_modules)
test:
	@echo "==> Running test suite..."
	@go test -v -race ./cmd/... ./internal/... ./pkg/...

# Verify compilation
build:
	@echo "==> Verifying project compilation..."
	@go build -o /dev/null ./cmd/gateway/...

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down
