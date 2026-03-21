package transaction

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

// TransactionTotalsInput is the Huma input for transaction totals by month.
type TransactionTotalsInput struct {
	StartMonth int `query:"startMonth" required:"true" minimum:"1" maximum:"12" doc:"Start month (1-12)"`
	StartYear  int `query:"startYear" required:"true" doc:"Start year"`
	EndMonth   int `query:"endMonth" required:"true" minimum:"1" maximum:"12" doc:"End month (1-12)"`
	EndYear    int `query:"endYear" required:"true" doc:"End year"`
}

// TransactionTotalsCategory is one category's sum within a single month.
type TransactionTotalsCategory struct {
	CategoryID string `json:"categoryID" doc:"Category UUID"`
	Total      string `json:"total" doc:"Sum of transaction amounts for the month"`
}

// TransactionTotalsMonth is one calendar month: totals are grouped by month first, then by category.
type TransactionTotalsMonth struct {
	Year       int                         `json:"year" doc:"Year"`
	Month      int                         `json:"month" doc:"Month (1-12)"`
	YearMonth  string                      `json:"yearMonth" doc:"Month as YYYY-MM (UTC calendar month)"`
	ByCategory []TransactionTotalsCategory `json:"byCategory" doc:"Totals for each category in this month"`
}

// TransactionTotalsResponseBody is the response body for transaction totals.
type TransactionTotalsResponseBody struct {
	// ByMonth is ordered by calendar month. Each entry is one month in the requested range.
	ByMonth []TransactionTotalsMonth `json:"byMonth" doc:"One object per month; months with no transactions have an empty byCategory array"`
}

// TransactionTotalsOutput is the Huma output for transaction totals.
type TransactionTotalsOutput struct {
	Body TransactionTotalsResponseBody
}

type transactionTotalsReader interface {
	TotalsByMonthAndCategory(ctx context.Context, startMonth, startYear, endMonth, endYear int) ([]transaction.MonthTotals, error)
}

// TransactionTotalsHandler handles GET /v1/transaction/totals.
type TransactionTotalsHandler struct {
	Reader transactionTotalsReader
}

// NewTransactionTotalsHandler creates a new TransactionTotalsHandler.
func NewTransactionTotalsHandler(reader transactionTotalsReader) *TransactionTotalsHandler {
	return &TransactionTotalsHandler{Reader: reader}
}

// Register registers the transaction totals endpoint with the Huma API.
func (h *TransactionTotalsHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "transaction-totals",
		Method:      http.MethodGet,
		Path:        "/v1/transaction/totals",
		Summary:     "Transaction totals by month",
		Description: "Returns totals grouped by calendar month, then by category (transaction_date in UTC). byMonth includes every month in the requested range; months with no transactions have an empty byCategory array (this is independent of budget table rows).",
		Tags:        []string{"Transactions"},
	}, h.handle)
}

func (h *TransactionTotalsHandler) handle(ctx context.Context, input *TransactionTotalsInput) (*TransactionTotalsOutput, error) {
	logData := logging.GetLogData(ctx)

	if input.EndYear < input.StartYear || (input.EndYear == input.StartYear && input.EndMonth < input.StartMonth) {
		return nil, huma.NewError(http.StatusBadRequest, "end date must be >= start date", nil)
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("transactionTotalsMs")
	}
	months, err := h.Reader.TotalsByMonthAndCategory(ctx, input.StartMonth, input.StartYear, input.EndMonth, input.EndYear)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to load transaction totals", err)
	}

	if months == nil {
		months = []transaction.MonthTotals{}
	}

	if logData != nil {
		logData.AddData("monthCount", len(months))
	}

	body := TransactionTotalsResponseBody{
		ByMonth: make([]TransactionTotalsMonth, len(months)),
	}
	for i, m := range months {
		// Non-nil empty slice so JSON serializes as [] not null when there are no categories.
		cats := make([]TransactionTotalsCategory, 0, len(m.Categories))
		for _, c := range m.Categories {
			cats = append(cats, TransactionTotalsCategory{
				CategoryID: c.CategoryID.String(),
				Total:      c.Total.String(),
			})
		}
		body.ByMonth[i] = TransactionTotalsMonth{
			Year:       m.Year,
			Month:      m.Month,
			YearMonth:  fmt.Sprintf("%04d-%02d", m.Year, m.Month),
			ByCategory: cats,
		}
	}

	return &TransactionTotalsOutput{Body: body}, nil
}
