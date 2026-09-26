package transaction

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeScanner feeds Scan destinations for scanListRow unit tests.
type fakeScanner struct {
	vals []any
}

func (f fakeScanner) Scan(dest ...any) error {
	for i, d := range dest {
		switch p := d.(type) {
		case *uuid.UUID:
			*p = f.vals[i].(uuid.UUID)
		case *uuid.NullUUID:
			*p = f.vals[i].(uuid.NullUUID)
		case *decimal.Decimal:
			*p = f.vals[i].(decimal.Decimal)
		case *string:
			*p = f.vals[i].(string)
		case *time.Time:
			*p = f.vals[i].(time.Time)
		case *sql.NullString:
			if f.vals[i] == nil {
				*p = sql.NullString{}
			} else {
				*p = sql.NullString{String: f.vals[i].(string), Valid: true}
			}
		case *int64:
			*p = f.vals[i].(int64)
		default:
			return fmt.Errorf("unsupported scan dest %T at %d", d, i)
		}
	}
	return nil
}

func TestScanListRow_ReadsTotalCountFromSQLColumn(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	accountID := uuid.Must(uuid.NewV4())
	created := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	scanner := fakeScanner{vals: []any{
		id,
		accountID,
		uuid.NullUUID{},
		decimal.NewFromInt(5),
		"Coffee",
		created,
		created,
		nil,
		int64(42),
	}}

	tx, totalCount, err := scanListRow(scanner)
	require.NoError(t, err)
	assert.Equal(t, id, tx.ID)
	assert.Equal(t, accountID, tx.AccountID)
	assert.Equal(t, "Coffee", tx.TransactionName)
	assert.Nil(t, tx.CategoryID)
	assert.Nil(t, tx.MerchantName)
	assert.Equal(t, int64(42), totalCount)
}

func sampleTx(createdAt time.Time) *Transaction {
	return &Transaction{
		ID:              uuid.Must(uuid.NewV4()),
		AccountID:       uuid.Must(uuid.NewV4()),
		Amount:          decimal.NewFromInt(10),
		TransactionName: "test",
		TransactionDate: createdAt,
		CreatedAt:       createdAt,
	}
}

func TestPageListResult_EmptyIncludesTotalCount(t *testing.T) {
	got := pageListResult(nil, 20, 0, nil, 0)
	assert.Nil(t, got.Transactions)
	assert.Nil(t, got.NextCursor)
	assert.Equal(t, 0, got.TotalCount)
}

func TestPageListResult_HasMoreSetsNextCursor(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	t2 := t0.Add(2 * time.Hour)
	// limit+1 probe page: 3 rows for limit 2
	rows := []*Transaction{sampleTx(t2), sampleTx(t1), sampleTx(t0)}

	got := pageListResult(rows, 2, 0, nil, 5)
	require.Len(t, got.Transactions, 2)
	require.NotNil(t, got.NextCursor)
	assert.Equal(t, 2, got.NextCursor.Position)
	assert.Equal(t, 2, got.NextCursor.Limit)
	assert.Equal(t, t2, got.NextCursor.MaxCreationTime)
	assert.Equal(t, 5, got.TotalCount)
}

func TestPageListResult_LastPageNoNextCursor(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []*Transaction{sampleTx(t0)}

	got := pageListResult(rows, 2, 2, nil, 3)
	require.Len(t, got.Transactions, 1)
	assert.Nil(t, got.NextCursor)
	assert.Equal(t, 3, got.TotalCount)
}

func TestPageListResult_FrozenMaxCreationTimeOnCursor(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	frozen := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	rows := []*Transaction{sampleTx(t0), sampleTx(t0), sampleTx(t0)}

	got := pageListResult(rows, 2, 4, &frozen, 10)
	require.NotNil(t, got.NextCursor)
	assert.Equal(t, 6, got.NextCursor.Position)
	assert.Equal(t, frozen, got.NextCursor.MaxCreationTime)
	assert.Equal(t, 10, got.TotalCount)
}

func TestPageListResult_OffsetPageMath(t *testing.T) {
	// Client page N=3, size S=25 → position 50; next page position 75 when has-more.
	t0 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]*Transaction, 26)
	for i := range rows {
		rows[i] = sampleTx(t0.Add(time.Duration(i) * time.Minute))
	}

	got := pageListResult(rows, 25, 50, nil, 100)
	require.Len(t, got.Transactions, 25)
	require.NotNil(t, got.NextCursor)
	assert.Equal(t, 75, got.NextCursor.Position)
	assert.Equal(t, 25, got.NextCursor.Limit)
	assert.Equal(t, 100, got.TotalCount)
}
