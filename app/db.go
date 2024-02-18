package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xwb1989/sqlparser"
)

var (
	SQLiteSignature = [16]byte{83, 81, 76, 105, 116, 101, 32, 102, 111, 114, 109, 97, 116, 32, 51, 0} // `SQLite format 3\000`
)

type DBLite struct {
	filePath string

	dbHeader
}

func New(filePath string) (*DBLite, error) {
	dbfile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file db: %s err=%w", filePath, err)
	}

	rawheader := make([]byte, HEADER_SIZE)
	if _, err := dbfile.Read(rawheader); err != nil {
		return nil, fmt.Errorf("read header got err=%w", err)
	}

	var header dbHeader
	err = binary.Read(bytes.NewReader(rawheader), binary.BigEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("parse header got err=%w", err)
	}

	if header.HeaderTitle != SQLiteSignature {
		return nil, fmt.Errorf("the file is not SQLite format.")
	}

	return &DBLite{
		filePath: filePath,
		dbHeader: header,
	}, nil
}

func (db *DBLite) OpenReader() (*os.File, error) {
	dbfile, err := os.Open(db.filePath)
	if err != nil {
		return nil, fmt.Errorf("open file db: %s err=%w", db.filePath, err)
	}

	return dbfile, nil
}

func (db *DBLite) PageHeaders() ([]PageHeader, error) {
	reader, err := db.OpenReader()
	if err != nil {
		return nil, err
	}

	result := []PageHeader{}
	for i := 0; i < int(db.PageCount); i++ {

		if i == 0 {
			// first page skip the DB header
			_, err = reader.Seek(HEADER_SIZE, io.SeekStart)
		} else {
			_, err = reader.Seek(int64(i)*int64(db.PageSize), io.SeekStart)
		}

		headerraw := make([]byte, 8)
		if _, err := reader.Read(headerraw); err != nil {
			return nil, fmt.Errorf("read page header err=%w", err)
		}

		// TODO: some got 12 bytes in the header

		var pageHeader dbPageHeader
		err = binary.Read(bytes.NewReader(headerraw), binary.BigEndian, &pageHeader)
		if err != nil {
			return nil, fmt.Errorf("parse page header err=%w", err)
		}

		result = append(result, PageHeader{
			PageIndex:    i,
			dbPageHeader: pageHeader,
		})
	}

	return result, nil
}

func (db *DBLite) GetTblSqlMaster() ([]*TblSqlMaster, error) {
	pageheaders, err := db.PageHeaders()
	if err != nil {
		return nil, err
	}

	// `sqlite_master` is in page 1
	numberOfCells := pageheaders[0].NumberOfCells

	reader, err := db.OpenReader()
	if err != nil {
		return nil, fmt.Errorf("open reader err=%w", err)
	}

	// skip header and page header
	_, err = reader.Seek(HEADER_SIZE+8, io.SeekStart)
	if err != nil {
		return nil, err
	}

	cellpointers := []uint16{}
	for i := 0; i < int(numberOfCells); i++ {
		twob := make([]byte, 2)
		if _, err := reader.Read(twob); err != nil {
			return nil, fmt.Errorf("read cell pointer err=%w", err)
		}

		var cp uint16
		if err := binary.Read(bytes.NewReader(twob), binary.BigEndian, &cp); err != nil {
			return nil, fmt.Errorf("parse 2 byte to uint16 err=%w", err)
		}

		cellpointers = append(cellpointers, cp)
	}

	result := []*TblSqlMaster{}

	for _, cp := range cellpointers {
		if _, err := reader.Seek(int64(cp), io.SeekStart); err != nil {
			return nil, err
		}

		numberOfBytesOfPayload, err := parseVarInt(reader)
		if err != nil {
			return nil, fmt.Errorf("parse number of bytes of payload err=%w", err)
		}
		_ = numberOfBytesOfPayload

		rowID, err := parseVarInt(reader)
		if err != nil {
			return nil, fmt.Errorf("parse row_id err=%w", err)
		}
		_ = rowID

		record, err := parseSQLMasterRecord(reader)
		if err != nil {
			return nil, fmt.Errorf("parse record table master err=%w", err)
		}

		result = append(result, record)
	}

	return result, nil
}
func (db *DBLite) CountRecordOfTable(tableName string) (int64, error) {
	tbl, err := db.getRecordSqlMaster(tableName)
	if err != nil {
		return 0, err
	}

	pageheaders, err := db.PageHeaders()
	if err != nil {
		return 0, fmt.Errorf("get page header err=%w", err)
	}

	header := pageheaders[tbl.RootPage-1]

	return int64(header.NumberOfCells), nil
}

func (db *DBLite) SelectColumn(columnName, tableName string, equalFilter map[string]interface{}) ([]string, error) {
	tbl, err := db.getRecordSqlMaster(tableName)
	if err != nil {
		return nil, err
	}

	// FIXME: remove this hack
	newddl := strings.ReplaceAll(tbl.SQL, "autoincrement", "")
	stmt, err := sqlparser.ParseStrictDDL(newddl)
	if err != nil {
		return nil, fmt.Errorf("parse DDL err=%w", err)
	}

	getColIdx := func(_stmt sqlparser.Statement, _colName string) int {
		for i, column := range _stmt.(*sqlparser.DDL).TableSpec.Columns {
			if strings.EqualFold(column.Name.String(), _colName) {
				return i
			}

		}
		return -1
	}

	colOrder := getColIdx(stmt, columnName)
	if colOrder == -1 {
		return nil, fmt.Errorf("doesn't have a column: %s on table: %s", columnName, tableName)
	}

	colCount := len(stmt.(*sqlparser.DDL).TableSpec.Columns)

	filter := &Filter{
		EqualFilter: []*EqualFilter{},
	}

	for k, v := range equalFilter {
		colID := getColIdx(stmt, k)
		if colID == -1 {
			return nil, fmt.Errorf("filter column_name: %s not found", k)
		}

		filter.EqualFilter = append(filter.EqualFilter, &EqualFilter{
			IndexColumn: colID,
			Value:       v,
			ValueType:   TypeString,
		})
	}

	reader, err := db.OpenReader()
	if err != nil {
		return nil, err
	}

	pageIdx := (tbl.RootPage - 1) * int64(db.PageSize)
	_, err = reader.Seek(pageIdx, io.SeekStart)
	if err != nil {
		return nil, err
	}

	headerraw := make([]byte, 8)
	if _, err := reader.Read(headerraw); err != nil {
		return nil, fmt.Errorf("read page header err=%w", err)
	}

	// TODO: some got 12 bytes in the header

	var pageHeader dbPageHeader
	err = binary.Read(bytes.NewReader(headerraw), binary.BigEndian, &pageHeader)
	if err != nil {
		return nil, fmt.Errorf("parse page header err=%w", err)
	}

	cellpointers := []uint16{}
	for i := 0; i < int(pageHeader.NumberOfCells); i++ {
		twob := make([]byte, 2)
		if _, err := reader.Read(twob); err != nil {
			return nil, fmt.Errorf("read cell pointer err=%w", err)
		}

		var cp uint16
		if err := binary.Read(bytes.NewReader(twob), binary.BigEndian, &cp); err != nil {
			return nil, fmt.Errorf("parse 2 byte to uint16 err=%w", err)
		}

		cellpointers = append(cellpointers, cp)
	}

	result := []string{}
	for _, cp := range cellpointers {
		if _, err := reader.Seek(pageIdx+int64(cp), io.SeekStart); err != nil {
			return nil, err
		}

		numberOfBytesOfPayload, err := parseVarInt(reader)
		if err != nil {
			return nil, fmt.Errorf("parse number of bytes of payload err=%w", err)
		}
		_ = numberOfBytesOfPayload

		rowID, err := parseVarInt(reader)
		if err != nil {
			return nil, fmt.Errorf("parse row_id err=%w", err)
		}
		_ = rowID

		// payload - parse header
		numBytesOfHeader, err := parseVarInt(reader)
		if err != nil {
			return nil, fmt.Errorf("parse number byte in header err=%w", err)
		}
		_ = numBytesOfHeader

		// parse serial type
		serialtypes := make([]uint32, colCount)
		for i := 0; i < colCount; i++ {
			tmp, err := parseVarInt(reader)
			if err != nil {
				return nil, fmt.Errorf("parse serial types got err=%w", err)
			}

			serialtypes[i] = tmp
		}

		// parse value
		recordvalues, err := parseColumnValue(reader, serialtypes, filter)
		if err != nil {
			return nil, fmt.Errorf("parse column values got err=%w", err)
		}

		if len(recordvalues) == 0 {
			continue
		}

		val, _ := recordvalues[colOrder].(string)
		result = append(result, val)
	}

	return result, nil
}

func (db *DBLite) getRecordSqlMaster(tableName string) (*TblSqlMaster, error) {
	records, err := db.GetTblSqlMaster()
	if err != nil {
		return nil, fmt.Errorf("get tbl sql master err=%w", err)
	}

	for _, record := range records {
		if record.TblName == tableName {
			return record, nil
		}
	}

	return nil, fmt.Errorf("not found table: %s\n", tableName)
}
