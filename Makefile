BINARY_NAME=gobankserver

build:
	@go build -o bin/${BINARY_NAME} .

run: build
	@./bin/${BINARY_NAME}

clean:
	 @go clean
	 @rm -rf bin