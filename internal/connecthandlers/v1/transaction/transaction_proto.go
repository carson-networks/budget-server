package v1Transaction

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	storagetransaction "github.com/carson-networks/budget-server/internal/storage/transaction"
)

func transactionToProto(tx *storagetransaction.Transaction) *transaction.Transaction {
	t := &transaction.Transaction{
		Id:              tx.ID.String(),
		AccountId:       tx.AccountID.String(),
		Amount:          tx.Amount.String(),
		TransactionName: tx.TransactionName,
		TransactionDate: timestamppb.New(tx.TransactionDate),
		CreatedAt:       timestamppb.New(tx.CreatedAt),
	}
	if tx.CategoryID != nil {
		cid := tx.CategoryID.String()
		t.CategoryId = &cid
	}
	if tx.MerchantName != nil {
		name := *tx.MerchantName
		t.MerchantName = &name
	}
	return t
}
