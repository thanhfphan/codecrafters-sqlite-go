package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	// Available if you need it!
	// "github.com/xwb1989/sqlparser"
)

const (
	HEADER_SIZE = 100
)

// Usage: your_sqlite3.sh sample.db .dbinfo
func main() {
	databaseFilePath := os.Args[1]
	command := os.Args[2]

	switch command {
	case ".dbinfo":
		databaseFile, err := os.Open(databaseFilePath)
		if err != nil {
			log.Fatal(err)
		}

		header := make([]byte, HEADER_SIZE)

		_, err = databaseFile.Read(header)
		if err != nil {
			log.Fatal(err)
		}

		var pageSize uint16
		if err := binary.Read(bytes.NewReader(header[16:18]), binary.BigEndian, &pageSize); err != nil {
			fmt.Println("Failed to read integer:", err)
			return
		}

		var totalTable uint16
		firstPage := make([]byte, pageSize-HEADER_SIZE) // only page 1 need to minus HEADER_SIZE
		_, err = databaseFile.Read(firstPage)
		if err != nil {
			log.Fatalf("read 1st page err: %v", err)
		}

		err = binary.Read(bytes.NewReader(firstPage[3:5]), binary.BigEndian, &totalTable)
		if err != nil {
			fmt.Println("read total table DB err: ", err)
			return
		}

		fmt.Printf("database page size: %v\n", pageSize)
		fmt.Printf("number of tables: %v\n", totalTable)
	default:
		fmt.Println("Unknown command", command)
		os.Exit(1)
	}
}
