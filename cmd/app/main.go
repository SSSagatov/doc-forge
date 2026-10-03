package main

import (
	"context"
	"flag"
	"log"

	"cloud-native-platform/db"
)

func main() {
	testDB := flag.Bool("test-db", false, "Check the PostgreSQL connection and exit")
	flag.Parse()

	if !*testDB {
		return
	}

	if err := db.TestDB(context.Background()); err != nil {
		log.Fatal(err)
	}

	log.Println("PostgreSQL connection test passed")
}
