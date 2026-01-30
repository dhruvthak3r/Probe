package logger

import (
	"log"
	"os"
)

var Log *log.Logger

func Init(path string) error {
	file, err := os.OpenFile(
		path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return err
	}

	Log = log.New(file, "", log.LstdFlags)
	return nil
}
