package model

import (
	"fmt"
	"strings"
)

// OrderUnit is how the customer buys: one piece, one bundle, or one dozen.
type OrderUnit string

const (
	OrderUnitPiece  OrderUnit = "piece"
	OrderUnitBundle OrderUnit = "bundle"
	OrderUnitDozen  OrderUnit = "dozen"
)

// DozenPieceCount is fixed for cleaning essentials.
const DozenPieceCount = 12

func (u OrderUnit) Valid() bool {
	switch u {
	case OrderUnitPiece, OrderUnitBundle, OrderUnitDozen:
		return true
	default:
		return false
	}
}

func ParseOrderUnit(s string) (OrderUnit, error) {
	u := OrderUnit(strings.TrimSpace(strings.ToLower(s)))
	if !u.Valid() {
		return "", fmt.Errorf("invalid unit %q", s)
	}
	return u, nil
}
