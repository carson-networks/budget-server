package account

// Account is the API response model for an account.
type Account struct {
	ID      string `json:"id" doc:"Account UUID"`
	Name    string `json:"name" doc:"Account name"`
	Type    int    `json:"type" doc:"Account type: 0=Cash, 1=Credit Cards, 2=Investments, 3=Loans, 4=Assets"`
	SubType string `json:"subType" doc:"Account sub-type"`
	Balance string `json:"balance" doc:"Decimal balance"`
}
