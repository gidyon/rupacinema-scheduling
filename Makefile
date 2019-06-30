SERVER_OUT := "server.bin"
CLIENT_OUT := "client.bin"
PKG := "github.com/gidyon/rupacinema/schedule"
SERVER_PKG_BUILD := "${PKG}/cmd/server"
CLIENT_PKG_BUILD := "${PKG}/cmd/client"

API_VERSION := "v1"
API_IN_PATH := "api/proto"

API_OUT_PATH := "pkg/api"
SWAGGER_DOC_OUT_PATH := "api/swagger"

gen_certs: ## Generate server certs for encryption
	@openssl req -x509 \
		-newkey rsa:1024 \
		-keyout certs/server.key \
		-out certs/server.crt \
		-days 365 \
		-nodes -subj '/CN=localhost'
		
gen_stub: ## Generate client stub in Go language
	@protoc -I=$(API_IN_PATH) -I=third_party -I=../ --go_out=plugins=grpc:$(API_OUT_PATH) schedule.proto

gen_rest: ## Generate reverse proxy server to for REST APIs to gRPC server
	@protoc -I=$(API_IN_PATH) -I=third_party -I=../ --grpc-gateway_out=logtostderr=true:$(API_OUT_PATH) schedule.proto

gen_swagger_doc: ## Generate swagger documentation
	@protoc -I=$(API_IN_PATH) -I=third_party -I=../ --swagger_out=logtostderr=true:$(SWAGGER_DOC_OUT_PATH) schedule.proto

gen_api_all: gen_stub gen_rest gen_swagger_doc ## Auto-generate grpc go sources
	
build_server_prod: ## Build a production binary for server
	@CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -a -installsuffix cgo -ldflags '-s' -v -o $(SERVER_OUT) $(SERVER_PKG_BUILD)

build_server: ## Build the binary file for server
	@go build -i -v -o $(SERVER_OUT) $(SERVER_PKG_BUILD)

build_client: ## Build the binary file for client
	@go build -i -v -o $(CLIENT_OUT) $(CLIENT_PKG_BUILD)

build_all:	build_server build_client # Build client and server binaries

clean_server: ## Remove server binary
	@rm $(SERVER_OUT)

clean_client: ## Remove client binary
	@rm $(CLIENT_OUT)

clean: ## Remove previous builds
	@rm $(SERVER_OUT) $(CLIENT_OUT) $(API_OUT)

docker_build: ## Create a docker image for the service
	docker build --no-cache -t rupacinema-schedule:dev .

docker_run: # Run the rupacinema-schedule:dev image
	docker run --rm -it -p 11093:11093 -e REDIS_URL=redis rupacinema-schedule:dev -uflag

docker_rmi: ## Remove docker image called rupacinema-schedule:dev
	docker rmi rupacinema-schedule:dev
	
run_app: clean_server build_server_prod docker_rmi docker_build docker_run

prepare_app: clean_server build_server_prod docker_rmi
	
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
