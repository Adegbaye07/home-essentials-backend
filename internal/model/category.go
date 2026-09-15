package model

import "fmt"

type Category string

const (
	CategoryCrossBody  Category = "cross_body"
	CategoryHobo       Category = "hobo"
	CategoryDuffel     Category = "duffel"
	CategoryMaleToilet Category = "male_toilet"
	CategorySchool     Category = "school"
	CategoryTravel     Category = "travel"
	CategoryLaptop     Category = "laptop"
	CategoryPurse      Category = "purse"
	CategoryClutch     Category = "clutch"
	CategoryTote       Category = "tote"
	CategoryShoulder   Category = "shoulder"
	CategoryShopping   Category = "shopping"
	CategoryRope       Category = "rope"
	CategorySatchel    Category = "satchel"
	CategoryJute       Category = "jute"
	CategoryLunchBox   Category = "lunch_box"
	CategoryWaistPurse Category = "waist_purse"
	CategoryFolder     Category = "folder"
	CategoryPencilCase Category = "pencil_case"
	CategoryHandBag    Category = "hand_bag"
	CategoryFlapBag    Category = "flap_bag"
	// CategoryMens is legacy; still valid so existing products load.
	CategoryMens Category = "mens"
)

func (c Category) Valid() bool {
	switch c {
	case CategoryCrossBody, CategoryHobo, CategoryDuffel, CategoryMaleToilet,
		CategorySchool, CategoryTravel, CategoryLaptop, CategoryPurse,
		CategoryClutch, CategoryTote, CategoryShoulder, CategoryShopping,
		CategoryRope, CategorySatchel, CategoryJute, CategoryLunchBox,
		CategoryWaistPurse, CategoryFolder, CategoryPencilCase, CategoryHandBag, CategoryFlapBag, CategoryMens:
		return true
	default:
		return false
	}
}

func ParseCategory(s string) (Category, error) {
	c := Category(s)
	if !c.Valid() {
		return "", fmt.Errorf("invalid category %q", s)
	}
	return c, nil
}
