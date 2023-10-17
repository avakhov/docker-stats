f:
	gofmt -w .

t:
	go mod tidy -v

r:
	reflex -s -- go run main.go
