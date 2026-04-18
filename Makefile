BINARY_NODE    = gaminium
BINARY_WALLET  = gmn-wallet
BINARY_MINE    = gmn-mine
BINARY_ORACLE  = gmn-oracle

BUILD_DIR = build

.PHONY: all build clean test lint install

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NODE)   ./cmd/gaminium/
	go build -o $(BUILD_DIR)/$(BINARY_WALLET) ./cmd/gmn-wallet/
	go build -o $(BUILD_DIR)/$(BINARY_MINE)   ./cmd/gmn-mine/
	go build -o $(BUILD_DIR)/$(BINARY_ORACLE) ./cmd/gmn-oracle/
	@echo "Build complete → $(BUILD_DIR)/"

test:
	go test ./tests/... -v

test-short:
	go test ./tests/... -short

lint:
	golangci-lint run ./...

vet:
	go vet ./...

install:
	go install ./cmd/gaminium/
	go install ./cmd/gmn-wallet/
	go install ./cmd/gmn-mine/
	go install ./cmd/gmn-oracle/

clean:
	rm -rf $(BUILD_DIR)
	go clean

node: build
	./$(BUILD_DIR)/$(BINARY_NODE)

wallet: build
	./$(BUILD_DIR)/$(BINARY_WALLET) new

mine: build
	@echo "Usage: make mine ADDR=GMN1..."
	./$(BUILD_DIR)/$(BINARY_MINE) $(ADDR)

oracle: build
	@echo "Usage: make oracle MNEMONIC='word1 ... word24' COUNTRY=US"
	./$(BUILD_DIR)/$(BINARY_ORACLE) "$(MNEMONIC)" "$(COUNTRY)"

tidy:
	go mod tidy

fmt:
	go fmt ./...
