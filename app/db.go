package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
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
		return nil, fmt.Errorf("open file db: %s err: %w", filePath, err)
	}

	rawheader := make([]byte, HEADER_SIZE)
	if _, err := dbfile.Read(rawheader); err != nil {
		return nil, fmt.Errorf("read header got err: %w", err)
	}

	var header dbHeader

	err = binary.Read(bytes.NewReader(rawheader), binary.BigEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("parse header got err: %w", err)
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
		return nil, fmt.Errorf("open file db: %s err: %w", db.filePath, err)
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
		rawdata := make([]byte, 8)
		var pageHeader dbPageHeader

		if i == 0 {
			// first page skip the DB header
			_, err = reader.Seek(HEADER_SIZE, io.SeekStart)
		} else {
			_, err = reader.Seek(int64(i)*int64(db.PageSize), io.SeekStart)
		}

		if _, err := reader.Read(rawdata); err != nil {
			return nil, fmt.Errorf("read page header err: %w", err)
		}

		// TODO: some got 12 bytes in the header

		err = binary.Read(bytes.NewReader(rawdata), binary.BigEndian, &pageHeader)
		if err != nil {
			return nil, fmt.Errorf("parse page header err: %w", err)
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
		return nil, fmt.Errorf("open reader err: %w", err)
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
			return nil, fmt.Errorf("read cell pointer err: %w", err)
		}

		var cp uint16
		if err := binary.Read(bytes.NewReader(twob), binary.BigEndian, &cp); err != nil {
			return nil, fmt.Errorf("parse 2 byte to uint16 err: %w", err)
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
			return nil, fmt.Errorf("parse number of bytes of payload err: %w", err)
		}
		_ = numberOfBytesOfPayload

		rowID, err := parseVarInt(reader)
		if err != nil {
			return nil, fmt.Errorf("parse row_id err: %w", err)
		}
		_ = rowID

		record, err := parseSQLMasterRecord(reader)
		if err != nil {
			return nil, fmt.Errorf("parse record table master err: %w", err)
		}

		result = append(result, record)
	}

	return result, nil
}

func (db *DBLite) CountRecordOfTable(tableName string) (int64, error) {
	records, err := db.GetTblSqlMaster()
	if err != nil {
		return 0, fmt.Errorf("get tbl sql master err: %w", err)
	}

	var tbl *TblSqlMaster
	for _, record := range records {
		if record.TblName == tableName {
			tbl = record
			break
		}
	}
	if tbl == nil {
		return 0, fmt.Errorf("not found table: %s\n", tableName)
	}

	pageheaders, err := db.PageHeaders()
	if err != nil {
		return 0, fmt.Errorf("get page header err: %w", err)
	}

	header := pageheaders[tbl.RootPage]

	return int64(header.NumberOfCells), nil
}
