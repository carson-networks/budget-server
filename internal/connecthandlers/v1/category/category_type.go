package v1Category

import (
	"fmt"

	gencategory "github.com/carson-networks/budget-server/internal/connecthandlers/gen/category"
	storagecategory "github.com/carson-networks/budget-server/internal/storage/category"
)

func StorageCategoryTypeFromProto(t gencategory.CategoryType) (storagecategory.CategoryType, error) {
	switch t {
	case gencategory.CategoryType_CATEGORY_TYPE_UNSPECIFIED:
		return 0, fmt.Errorf("category type must be specified")
	case gencategory.CategoryType_CATEGORY_TYPE_INCOME:
		return storagecategory.CatergoryType_Income, nil
	case gencategory.CategoryType_CATEGORY_TYPE_EXPENSE:
		return storagecategory.CatergoryType_Expense, nil
	default:
		return 0, fmt.Errorf("invalid category type: %v", t)
	}
}

func ProtoCategoryTypeFromStorage(t storagecategory.CategoryType) gencategory.CategoryType {
	switch t {
	case storagecategory.CatergoryType_Income:
		return gencategory.CategoryType_CATEGORY_TYPE_INCOME
	case storagecategory.CatergoryType_Expense:
		return gencategory.CategoryType_CATEGORY_TYPE_EXPENSE
	default:
		return gencategory.CategoryType_CATEGORY_TYPE_UNSPECIFIED
	}
}
