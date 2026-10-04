.PHONY: test run migrate

test:
	go test -v -count=1 ./db -run "^TestDB$$"
run:
	go run cmd/app/main.go 
migrate:
	go run cmd/app/migrator/main.go
