package main

const (
	HEADER_SIZE = 100
)

const (
	INTERIOR_INDEX_PAGE = 0x02
	INTERIOR_TABLE_PAGE = 0x05
	LEAF_INDEX_PAGE     = 0x0a
	LEAF_TABLE_PAGE     = 0x0d
)

const (
	MASK_FIST_BIT_ENABLE = 0b1000_0000
	MASK_LAST_SEVEN_BIT  = 0b0111_1111
)

type ValueType string

const (
	TypeString ValueType = "string"
)
