package operator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage"
)

func TestNewOperatorDelegator_NumWorkersLessThanOne_DefaultsToOne(t *testing.T) {
	mockStorage := &MockIStorage{}
	d := NewOperatorDelegator(mockStorage, 0)
	require.NotNil(t, d)
	d.Start()
	defer d.Stop()

	// With 1 worker, Process should succeed
	tx := &mockTx{}
	writer := storage.NewWriterForTest(tx, &storage.MockIAccountWriter{}, &storage.MockITransactionWriter{})
	mockStorage.EXPECT().
		Write(mock.Anything).
		Return(&writer, nil)

	mockAction := &actions.MockIAction{}
	mockAction.EXPECT().
		Perform(mock.Anything, &writer).
		Return(nil)

	err := d.Process(context.Background(), mockAction)
	require.NoError(t, err)
	mockStorage.AssertExpectations(t)
	mockAction.AssertExpectations(t)
}

func TestOperatorDelegator_Process_Success(t *testing.T) {
	mockStorage := &MockIStorage{}
	d := NewOperatorDelegator(mockStorage, 2)
	d.Start()
	defer d.Stop()

	tx := &mockTx{}
	writer := storage.NewWriterForTest(tx, &storage.MockIAccountWriter{}, &storage.MockITransactionWriter{})
	mockStorage.EXPECT().
		Write(mock.Anything).
		Return(&writer, nil)

	mockAction := &actions.MockIAction{}
	mockAction.EXPECT().
		Perform(mock.Anything, &writer).
		Return(nil)

	err := d.Process(context.Background(), mockAction)
	require.NoError(t, err)
	assert.True(t, tx.commitCalled)
	mockStorage.AssertExpectations(t)
	mockAction.AssertExpectations(t)
}

func TestOperatorDelegator_Process_Error(t *testing.T) {
	performErr := errors.New("action failed")
	mockStorage := &MockIStorage{}
	d := NewOperatorDelegator(mockStorage, 1)
	d.Start()
	defer d.Stop()

	tx := &mockTx{}
	writer := storage.NewWriterForTest(tx, &storage.MockIAccountWriter{}, &storage.MockITransactionWriter{})
	mockStorage.EXPECT().
		Write(mock.Anything).
		Return(&writer, nil)

	mockAction := &actions.MockIAction{}
	mockAction.EXPECT().
		Perform(mock.Anything, &writer).
		Return(performErr)

	err := d.Process(context.Background(), mockAction)
	assert.ErrorIs(t, err, performErr)
	assert.True(t, tx.rollbackCalled)
	mockStorage.AssertExpectations(t)
	mockAction.AssertExpectations(t)
}

func TestOperatorDelegator_Process_ContextCancelled(t *testing.T) {
	mockStorage := &MockIStorage{}
	d := NewOperatorDelegator(mockStorage, 1)
	d.Start()
	defer d.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockAction := &actions.MockIAction{}

	// Process returns immediately when context is already cancelled.
	// The worker may still process the item in the background, so we allow
	// mock calls with Maybe() to avoid panics from unexpected calls.
	tx := &mockTx{}
	writer := storage.NewWriterForTest(tx, &storage.MockIAccountWriter{}, &storage.MockITransactionWriter{})
	mockStorage.On("Write", mock.Anything).Maybe().Return(&writer, nil)
	mockAction.On("Perform", mock.Anything, mock.Anything).Maybe().Return(nil)

	err := d.Process(ctx, mockAction)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestOperatorDelegator_Stop_IsIdempotent(t *testing.T) {
	mockStorage := &MockIStorage{}
	d := NewOperatorDelegator(mockStorage, 1)
	d.Start()

	d.Stop()
	d.Stop() // Second call should not panic
}
