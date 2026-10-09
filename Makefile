.PHONY: test run migrate

test:
	go test -v -count=1 ./api-service/database/postgres -run "^TestDB$$"
	go test -v -count=1 ./api-service/database/repository -run "Test"
	go test -v -count=1 ./api-service/usecases -run "Test"
	go test -v -count=1 ./api-service/gemini -run "Test"
run:
	go run api-service/cmd/main.go 
migrate:
	go run api-service/cmd/migrator/main.go
