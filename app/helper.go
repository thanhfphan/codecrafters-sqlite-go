package main

import (
	"fmt"
	"os"
)

func parseVarInt(reader *os.File) (uint32, error) {
	usablebytes, err := readVarIntLength(reader)
	if err != nil {
		return 0, fmt.Errorf("read varint err: %w", err)
	}

	var result uint32

	for idx, b := range usablebytes {
		usablebit := uint32(7)
		if idx == 8 {
			usablebit = 8 // 9th byte use all bit
		}

		var usablevalue uint8
		if usablebit == 7 {
			usablevalue = b & MASK_LAST_SEVEN_BIT
		} else {
			usablevalue = b
		}

		result = (result << usablebit) | uint32(usablevalue)
	}

	return result, nil
}

func readVarIntLength(reader *os.File) ([]byte, error) {
	var result []byte

	for i := 0; i < 9; i++ {
		tmp := make([]byte, 1)
		if _, err := reader.Read(tmp); err != nil {
			return nil, err
		}

		result = append(result, tmp[0])

		if tmp[0]&MASK_FIST_BIT_ENABLE == 0 {
			break
		}
	}

	return result, nil
}

// https://www.sqlite.org/fileformat.html#record_format
func parseSQLMasterRecord(reader *os.File) (*TblSqlMaster, error) {
	numBytesOfHeader, err := parseVarInt(reader)
	if err != nil {
		return nil, fmt.Errorf("parse number byte in header err: %w", err)
	}
	_ = numBytesOfHeader

	// CREATE TABLE sqlite_schema(
	//   type text,
	//   name text,
	//   tbl_name text,
	//   rootpage integer,
	//   sql text
	// );
	columnCount := 5

	serialtypes := make([]uint32, columnCount)
	for i := 0; i < columnCount; i++ {
		tmp, err := parseVarInt(reader)
		if err != nil {
			return nil, fmt.Errorf("parse serial types got err: %w", err)
		}

		serialtypes[i] = tmp
	}

	recordvalues, err := parseColumnValue(reader, serialtypes)
	if err != nil {
		return nil, fmt.Errorf("parse column values got err: %w", err)
	}

	record := &TblSqlMaster{}
	// type
	tmp, _ := recordvalues[0].(string)
	record.Type = tmp
	// name
	tmp, _ = recordvalues[1].(string)
	record.Name = tmp
	// table name
	tmp, _ = recordvalues[2].(string)
	record.TblName = tmp
	// root page
	tmpint, _ := ParseInt64(recordvalues[3])
	record.RootPage = tmpint
	// SQL DDL
	tmp, _ = recordvalues[4].(string)
	record.SQL = tmp

	return record, nil
}

// https://www.sqlite.org/fileformat.html#record_format
func parseColumnValue(reader *os.File, serialtypes []uint32) ([]interface{}, error) {
	result := make([]interface{}, len(serialtypes))
	for i, st := range serialtypes {
		if st == 0 {
			result[i] = nil
		} else if st == 1 || st == 2 || st == 3 || st == 4 {
			tmp := make([]byte, st)
			if _, err := reader.Read(tmp); err != nil {
				return nil, fmt.Errorf("read serial_type: %d, err: %w", st, err)
			}

			result[i], _ = ParseInt(tmp)
		} else if st == 5 {
			tmp := make([]byte, 6)
			if _, err := reader.Read(tmp); err != nil {
				return nil, fmt.Errorf("read serial_type: %d, err: %w", st, err)
			}

			result[i], _ = ParseInt(tmp)
		} else if st == 6 {
			tmp := make([]byte, 8)
			if _, err := reader.Read(tmp); err != nil {
				return nil, fmt.Errorf("read serial_type: %d, err: %w", st, err)
			}

			result[i], _ = ParseInt(tmp)
		} else if st == 8 {
			result[i] = 0
		} else if st == 9 {
			result[i] = 1
		} else if st == 10 || st == 11 {
			result[i] = nil
		} else if st >= 13 && st%2 == 1 {
			nbytes := (st - 13) / 2
			rawdata := make([]byte, nbytes)

			if _, err := reader.Read(rawdata); err != nil {
				return nil, fmt.Errorf("read serial_type: %d, err: %w", st, err)
			}

			result[i] = string(rawdata)

		} else if st >= 12 && st%2 == 0 {
			nbytes := (st - 12) / 2
			rawdata := make([]byte, nbytes)

			if _, err := reader.Read(rawdata); err != nil {
				return nil, fmt.Errorf("read serial_type: %d, err: %w", st, err)
			}

			result[i] = rawdata
		} else {
			return nil, fmt.Errorf("not support serial_type: %d", st)
		}
	}

	return result, nil
}

// https://stackoverflow.com/questions/32096647/variable-length-twos-complement-to-int64
func ParseInt(b []byte) (int64, error) {
	if len(b) > 8 {
		return 0, fmt.Errorf("value does not fit in a int64")
	}

	var n int64
	for i, v := range b {
		shift := uint((len(b) - i - 1) * 8)
		if i == 0 && v&0x80 != 0 {
			n -= 0x80 << shift
			v &= 0x7f
		}
		n += int64(v) << shift
	}
	return n, nil
}

func ParseInt64(n interface{}) (int64, error) {
	switch i := n.(type) {
	case uint8:
		return int64(i), nil
	case uint16:
		return int64(i), nil
	case uint32:
		return int64(i), nil
	case int:
		return int64(i), nil
	case int32:
		return int64(i), nil
	case int64:
		return i, nil
	}

	return 0, fmt.Errorf("cant parse to int64: %v", n)
}
