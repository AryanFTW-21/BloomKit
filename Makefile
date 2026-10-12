.PHONY: test race bench fp compare vet

test:
	go test ./...

race:
	go test -race ./...

bench:
	go test ./bloom/ -run=^$$ -bench=. -benchmem

fp:
	go run ./cmd/bloomkit fp

compare:
	go run ./cmd/bloomkit compare

vet:
	go vet ./...
