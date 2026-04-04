package v1Category

import (
	"fmt"

	category "github.com/carson-networks/budget-server/internal/connecthandlers/gen/category/v1"
	storagecategory "github.com/carson-networks/budget-server/internal/storage/category"
)

// FromConnectCategoryType maps protobuf CategoryType from Connect requests into storage.
// CATEGORY_TYPE_UNSPECIFIED and unknown values return an error.
func FromConnectCategoryType(t category.CategoryType) (storagecategory.CategoryType, error) {
	switch t {
	case category.CategoryType_CATEGORY_TYPE_UNSPECIFIED:
		return 0, fmt.Errorf("category type must be specified")
	case category.CategoryType_CATEGORY_TYPE_INCOME:
		return storagecategory.CatergoryType_Income, nil
	case category.CategoryType_CATEGORY_TYPE_EXPENSE:
		return storagecategory.CatergoryType_Expense, nil
	default:
		return 0, fmt.Errorf("invalid category type: %v", t)
	}
}

func toConnectCategoryType(t storagecategory.CategoryType) category.CategoryType {
	switch t {
	case storagecategory.CatergoryType_Income:
		return category.CategoryType_CATEGORY_TYPE_INCOME
	case storagecategory.CatergoryType_Expense:
		return category.CategoryType_CATEGORY_TYPE_EXPENSE
	default:
		return category.CategoryType_CATEGORY_TYPE_UNSPECIFIED
	}
}
