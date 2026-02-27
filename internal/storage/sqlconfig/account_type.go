package sqlconfig

type AccountType int8

const (
	AccountTypeCash AccountType = iota
	AccountTypeCreditCards
	AccountTypeInvestments
	AccountTypeLoans
	AccountTypeAssets
)
