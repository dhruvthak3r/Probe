package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	Pool *sql.DB
}

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

func getDBConfig() *DBConfig {
	return &DBConfig{
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Name:     os.Getenv("DB_NAME"),
	}
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
	)
}

func NewConnection() (*DB, error) {
	cfg := getDBConfig()

	conn, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)
	conn.SetConnMaxIdleTime(3 * time.Minute)

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	fmt.Println("conn est")

	return &DB{
		Pool: conn,
	}, nil
}
