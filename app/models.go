package main

// Offset	Size	Description
// 0		16		The header string: "SQLite format 3\000"
// 16		2		The database page size in bytes. Must be a power of two between 512 and 32768 inclusive, or the value 1 representing a page size of 65536.
// 18		1		File format write version. 1 for legacy; 2 for WAL.
// 19		1		File format read version. 1 for legacy; 2 for WAL.
// 20		1		Bytes of unused "reserved" space at the end of each page. Usually 0.
// 21		1		Maximum embedded payload fraction. Must be 64.
// 22		1		Minimum embedded payload fraction. Must be 32.
// 23		1		Leaf payload fraction. Must be 32.
// 24		4		File change counter.
// 28		4		Size of the database file in pages. The "in-header database size".
// 32		4		Page number of the first freelist trunk page.
// 36		4		Total number of freelist pages.
// 40		4		The schema cookie.
// 44		4		The schema format number. Supported schema formats are 1, 2, 3, and 4.
// 48		4		Default page cache size.
// 52		4		The page number of the largest root b-tree page when in auto-vacuum or incremental-vacuum modes, or zero otherwise.
// 56		4		The database text encoding. A value of 1 means UTF-8. A value of 2 means UTF-16le. A value of 3 means UTF-16be.
// 60		4		The "user version" as read and set by the user_version pragma.
// 64		4		True (non-zero) for incremental-vacuum mode. False (zero) otherwise.
// 68		4		The "Application ID" set by PRAGMA application_id.
// 72		20		Reserved for expansion. Must be zero.
// 92		4		The version-valid-for number.
// 96		4		SQLITE_VERSION_NUMBER
type dbHeader struct {
	HeaderTitle [16]byte
	PageSize    uint16
	_           [10]byte
	PageCount   uint32 // Size of the database file in pages. The "in-header database size".
	_           [68]byte
}

// The b-tree page header is 8 bytes in size for leaf pages and 12 bytes for interior pages
// Offset	Size	Description
// 0		1		The one-byte flag at offset 0 indicating the b-tree page type.
//   - A value of 2 (0x02) means the page is an interior index b-tree page.
//   - A value of 5 (0x05) means the page is an interior table b-tree page.
//   - A value of 10 (0x0a) means the page is a leaf index b-tree page.
//   - A value of 13 (0x0d) means the page is a leaf table b-tree page.
//     Any other value for the b-tree page type is an error.
//
// 1		2		The two-byte integer at offset 1 gives the start of the first freeblock on the page, or is zero if there are no freeblocks.
// 3		2		The two-byte integer at offset 3 gives the number of cells on the page.
// 5		2		The two-byte integer at offset 5 designates the start of the cell content area. A zero value for this integer is interpreted as 65536.
// 7		1		The one-byte integer at offset 7 gives the number of fragmented free bytes within the cell content area.
// 8		4		The four-byte page number at offset 8 is the right-most pointer. This value appears in the header of interior b-tree pages only and is omitted from all other pages.
type dbPageHeader struct {
	Type             uint8
	StartFreeBlock   uint16
	NumberOfCells    uint16
	StartContentArea uint16
	_                [1]byte
}

type PageHeader struct {
	PageIndex int
	dbPageHeader
}

// CREATE TABLE sqlite_schema(
//
//	type text,
//	name text,
//	tbl_name text,
//	rootpage integer,
//	sql text
//
// );
// https://www.sqlite.org/fileformat.html#storage_of_the_sql_database_schema
type TblSqlMaster struct {
	Type     string
	Name     string
	TblName  string
	RootPage int64
	SQL      string
}

type Filter struct {
	EqualFilter []*EqualFilter
}

type EqualFilter struct {
	IndexColumn int
	Value       interface{}
	ValueType   ValueType
}
