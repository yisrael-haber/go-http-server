server-build:
	@go build -o build/main main.go 

server-run:
	@go run main.go

server-help:
	@./build/main help