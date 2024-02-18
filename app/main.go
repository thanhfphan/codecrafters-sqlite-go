package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xwb1989/sqlparser"
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
		st, err := sqlparser.Parse(command)
		if err != nil {
			log.Fatalf("failed to parse command %s\n", command)
		}

		selectCmd, ok := st.(*sqlparser.Select)
		if ok {
			tableName := selectCmd.From[0].(*sqlparser.AliasedTableExpr).Expr.(sqlparser.TableName).Name.String()
			selectExpr := selectCmd.SelectExprs[0].(*sqlparser.AliasedExpr).Expr
			switch selectExpr := selectExpr.(type) {
			case *sqlparser.FuncExpr:
				funcName := selectExpr.Name.Lowered()
				if funcName != "count" {
					log.Fatal("only support select count")
				}

				count, err := dblite.CountRecordOfTable(tableName)
				if err != nil {
					log.Fatalf("count record table: %s, err: %v\n", tableName, err)
				}

				fmt.Println(count)
				os.Exit(0)
			case *sqlparser.ColName:
				targetCol := selectExpr.Name.Lowered()
				data, err := dblite.SelectColumn(targetCol, tableName)
				if err != nil {
					log.Fatalf("select column: %s from table: %s err: %v", targetCol, tableName, err)
				}

				for _, item := range data {
					fmt.Println(item)
				}

				os.Exit(0)
			}

			fmt.Println("Unknown command", command)
			os.Exit(1)
		}
	}
}
