package v1Account

import (
	"fmt"

	genaccount "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account"
	storageaccount "github.com/carson-networks/budget-server/internal/storage/account"
)

func FromConnectAccountType(t genaccount.AccountType) (storageaccount.AccountType, error) {
	switch t {
	case genaccount.AccountType_ACCOUNT_TYPE_UNSPECIFIED:
		return 0, fmt.Errorf("account type must be specified")
	case genaccount.AccountType_ACCOUNT_TYPE_CASH:
		return storageaccount.AccountTypeCash, nil
	case genaccount.AccountType_ACCOUNT_TYPE_CREDIT_CARDS:
		return storageaccount.AccountTypeCreditCards, nil
	default:
		return 0, fmt.Errorf("invalid account type: %v", t)
	}
}

func toConnectAccountType(t storageaccount.AccountType) genaccount.AccountType {
	switch t {
	case storageaccount.AccountTypeCash:
		return genaccount.AccountType_ACCOUNT_TYPE_CASH
	case storageaccount.AccountTypeCreditCards:
		return genaccount.AccountType_ACCOUNT_TYPE_CREDIT_CARDS
	default:
		return genaccount.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	}
}
