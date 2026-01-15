package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func NewConn() (*sql.DB, error) {
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
	return conn, nil
}
