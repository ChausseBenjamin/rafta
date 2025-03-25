BUILD_DIR=./build
APP=rafta

all: setup codegen clean protoset compile

setup:
	git submodule update --init

codegen: setup
	sqlc generate
	protoc \
		--proto_path=resources \
		--proto_path=external \
		--go_out=pkg/model \
		--go_opt=paths=source_relative \
		--go-grpc_out=pkg/model \
		--go-grpc_opt=paths=source_relative \
		resources/schema.proto

protoset: setup
	mkdir -p $(BUILD_DIR)
	protoc \
		--proto_path=resources \
		--proto_path=external \
		--descriptor_set_out=$(BUILD_DIR)/schema.protoset \
		--include_imports \
		resources/schema.proto

compile: codegen
	mkdir -p $(BUILD_DIR)
	go run ./internal/autogen > $(BUILD_DIR)/$(APP).1
	go build -o $(BUILD_DIR)/$(APP) .

clean:
	rm -rf $(BUILD_DIR)

.PHONY: run
run: setup codegen
	./resources/local_dev.sh

