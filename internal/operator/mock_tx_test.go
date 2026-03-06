package operator

import "context"

// mockTx implements the storage.txRunner interface for testing.
// It records whether Commit and Rollback were called and returns configurable errors.
type mockTx struct {
	commitErr      error
	rollbackErr    error
	commitCalled   bool
	rollbackCalled bool
}

func (m *mockTx) Commit(ctx context.Context) error {
	m.commitCalled = true
	return m.commitErr
}

func (m *mockTx) Rollback(ctx context.Context) error {
	m.rollbackCalled = true
	return m.rollbackErr
}
