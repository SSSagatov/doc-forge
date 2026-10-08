.PHONY: test run migrate

test:
	go test -v -count=1 ./database/postgres -run "^TestDB$$"
run:
	go run api-service/cmd/app/main.go 
migrate:
	go run cmd/app/migrator/main.go
