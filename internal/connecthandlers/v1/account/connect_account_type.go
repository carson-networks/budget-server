package v1Account

import (
	"fmt"

	account "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account/v1"
	storageaccount "github.com/carson-networks/budget-server/internal/storage/account"
)

// FromConnectAccountType maps protobuf AccountType from Connect requests into storage.
// Only CASH and CREDIT_CARDS are defined on the wire; UNSPECIFIED and unknown values error.
func FromConnectAccountType(t account.AccountType) (storageaccount.AccountType, error) {
	switch t {
	case account.AccountType_ACCOUNT_TYPE_UNSPECIFIED:
		return 0, fmt.Errorf("account type must be specified")
	case account.AccountType_ACCOUNT_TYPE_CASH:
		return storageaccount.AccountTypeCash, nil
	case account.AccountType_ACCOUNT_TYPE_CREDIT_CARDS:
		return storageaccount.AccountTypeCreditCards, nil
	default:
		return 0, fmt.Errorf("invalid account type: %v", t)
	}
}

func toConnectAccountType(t storageaccount.AccountType) account.AccountType {
	switch t {
	case storageaccount.AccountTypeCash:
		return account.AccountType_ACCOUNT_TYPE_CASH
	case storageaccount.AccountTypeCreditCards:
		return account.AccountType_ACCOUNT_TYPE_CREDIT_CARDS
	default:
		return account.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	}
}
