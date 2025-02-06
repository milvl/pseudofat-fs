BINARY_NAME=myfs
SRC_DIR=src
BUILD_DIR=../bin
# GO_FILES=$(shell find $(SRC_DIR) -type f -name '*.go')

all: build

build:
	mkdir -p $(BUILD_DIR)
	cd src/ && go build -o ../bin/$(BINARY_NAME) main.go && cd ..

clean:
	rm -rf $(BUILD_DIR)

.PHONY: all build clean
