package model

import "fmt"

type Category string

const (
	CategoryFootMats           Category = "foot_mats"
	CategoryDoorMats           Category = "door_mats"
	CategoryCenterMats         Category = "center_mats"
	CategoryRugs               Category = "rugs"
	CategoryCleaningEssentials Category = "cleaning_essentials"
)

func (c Category) Valid() bool {
	switch c {
	case CategoryFootMats, CategoryDoorMats, CategoryCenterMats, CategoryRugs, CategoryCleaningEssentials:
		return true
	default:
		return false
	}
}

func (c Category) IsCleaning() bool {
	return c == CategoryCleaningEssentials
}

func ParseCategory(s string) (Category, error) {
	c := Category(s)
	if !c.Valid() {
		return "", fmt.Errorf("invalid category %q", s)
	}
	return c, nil
}
