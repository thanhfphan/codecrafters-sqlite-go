package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	// Available if you need it!
)

// Usage: your_sqlite3.sh sample.db .dbinfo
func main() {
	databaseFilePath := os.Args[1]
	command := os.Args[2]

	dblite, err := New(databaseFilePath)
	if err != nil {
		log.Fatal(err)
	}

	switch command {
	case ".dbinfo":
		pageheaders, err := dblite.PageHeaders()
		if err != nil {
			log.Fatal("get all page header err: ", err)
		}

		fmt.Printf("database page size: %v\n", dblite.PageSize)
		fmt.Printf("number of tables: %v\n", pageheaders[0].NumberOfCells) // page 1 which is always a table b-tree page

	case ".tables":
		records, err := dblite.GetTblSqlMaster()
		if err != nil {
			log.Fatal(err)
		}

		tables := []string{}
		for _, item := range records {
			tables = append(tables, item.TblName)
		}

		fmt.Println(strings.Join(tables, " "))

	default:
		strs := strings.Split(command, " ")
		table := strs[len(strs)-1]

		count, err := dblite.CountRecordOfTable(table)
		if err != nil {
			log.Fatalf("count record table: %s, err: %w\n", table, err)
		}

		fmt.Print(count)

		// fmt.Println("Unknown command", command)
		// os.Exit(1)
	}
}
