.PHONY: test

test:
	go run cmd/app/main.go --test-db
run:
	go run cmd/app/main.go 
