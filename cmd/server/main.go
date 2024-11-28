package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Essen-Labs/bloom-be/pkg/app"
	_ "github.com/lib/pq"
)

func main() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := "bloom"

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s')", dbName)
	err = db.QueryRow(query).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	if !exists {
		// Create the target database
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			log.Fatalf("Failed to create database %s: %v", dbName, err)
		}
		fmt.Printf("Database %s created successfully\n", dbName)
	} else {
		fmt.Printf("Database %s already exists\n", dbName)
	}

	connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the new database: %v", err)
	}
	defer db.Close()

	fmt.Println("Successfully connected to the database and running app!")
	app.LoadApp(db).Run()
}
