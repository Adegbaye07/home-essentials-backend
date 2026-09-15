package model

import "fmt"

type SizeCode string

const (
	SizeS   SizeCode = "S"
	SizeM   SizeCode = "M"
	SizeL   SizeCode = "L"
	SizeXL  SizeCode = "XL"
	SizeXXL SizeCode = "XXL"
)

func (s SizeCode) Valid() bool {
	switch s {
	case SizeS, SizeM, SizeL, SizeXL, SizeXXL:
		return true
	default:
		return false
	}
}

func ParseSizeCode(s string) (SizeCode, error) {
	code := SizeCode(s)
	if !code.Valid() {
		return "", fmt.Errorf("invalid size %q", s)
	}
	return code, nil
}
