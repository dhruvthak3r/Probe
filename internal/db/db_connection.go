package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type DB struct {
	Pool *sql.DB
}

func NewConnection() (*DB, error) {
	env_err := godotenv.Load()
	if env_err != nil {
		fmt.Printf("Error loading .env file")
	}

	password := os.Getenv("db_password")
	conn, err := sql.Open("mysql", password)

	if err != nil {
		return nil, err
	}

	fmt.Println("connection est")
	return &DB{Pool: conn}, nil
}
