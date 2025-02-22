BUILD_DIR=./build
APP=rafta

all: setup codegen clean compile

setup:
	git submodule update --init

codegen:
	protoc \
		--proto_path=resources \
		--proto_path=external \
		--go_out=pkg/model \
		--go_opt=paths=source_relative \
		--go-grpc_out=pkg/model \
		--go-grpc_opt=paths=source_relative \
		resources/schema.proto

clean:
	rm -rf $(BUILD_DIR) || exit 1

compile:
	mkdir -p $(BUILD_DIR) || exit 1
	CGO_ENABLED=1 go run ./internal/manualgen > $(BUILD_DIR)/$(APP).1
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(APP) .

.PHONY: run
run:
	./resources/local_dev.sh

