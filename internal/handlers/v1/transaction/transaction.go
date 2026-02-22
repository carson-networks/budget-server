package transaction

// Transaction is the API response model for a transaction.
// It is used only for responses, not for request bodies.
type Transaction struct {
	ID              string `json:"id" doc:"Transaction UUID"`
	AccountID       string `json:"accountID" doc:"Account UUID"`
	CategoryID      string `json:"categoryID" doc:"Category UUID"`
	Amount          string `json:"amount" doc:"Decimal amount"`
	TransactionName string `json:"transactionName" doc:"Name of the transaction"`
	TransactionDate string `json:"transactionDate" doc:"RFC3339 transaction date"`
}
