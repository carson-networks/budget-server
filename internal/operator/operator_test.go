package operator

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage"
)

func TestOperator_processItem_Success(t *testing.T) {
	ctx := context.Background()
	mockStorage := &MockIStorage{}
	mockAction := &actions.MockIAction{}

	tx := &mockTx{}
	wt := storage.NewWriterForTestWithTx(tx)

	mockStorage.EXPECT().
		Write(mock.Anything).
		Return(wt, nil)
	mockAction.EXPECT().
		Perform(mock.Anything, wt).
		Return(nil)

	queue := make(chan ActionItem, 1)
	op := NewOperator(mockStorage, queue)

	respCh := make(chan ActionItemResponse, 1)
	queue <- ActionItem{ctx: ctx, action: mockAction, response: respCh}
	close(queue)

	op.Run()

	resp := <-respCh
	require.NoError(t, resp.err)
	assert.True(t, tx.commitCalled, "Commit should have been called")
	assert.False(t, tx.rollbackCalled, "Rollback should not have been called")
	mockStorage.AssertExpectations(t)
	mockAction.AssertExpectations(t)
}

func TestOperator_processItem_WriteFails(t *testing.T) {
	ctx := context.Background()
	writeErr := errors.New("write failed")
	mockStorage := &MockIStorage{}
	mockAction := &actions.MockIAction{}

	mockStorage.EXPECT().
		Write(mock.Anything).
		Return(nil, writeErr)

	queue := make(chan ActionItem, 1)
	op := NewOperator(mockStorage, queue)

	respCh := make(chan ActionItemResponse, 1)
	queue <- ActionItem{ctx: ctx, action: mockAction, response: respCh}
	close(queue)

	op.Run()

	resp := <-respCh
	assert.ErrorIs(t, resp.err, writeErr)
	mockStorage.AssertExpectations(t)
	mockAction.AssertNotCalled(t, "Perform")
}

func TestOperator_processItem_PerformFails(t *testing.T) {
	ctx := context.Background()
	performErr := errors.New("perform failed")
	mockStorage := &MockIStorage{}
	mockAction := &actions.MockIAction{}

	tx := &mockTx{}
	wt := storage.NewWriterForTestWithTx(tx)

	mockStorage.EXPECT().
		Write(mock.Anything).
		Return(wt, nil)
	mockAction.EXPECT().
		Perform(mock.Anything, wt).
		Return(performErr)

	queue := make(chan ActionItem, 1)
	op := NewOperator(mockStorage, queue)

	respCh := make(chan ActionItemResponse, 1)
	queue <- ActionItem{ctx: ctx, action: mockAction, response: respCh}
	close(queue)

	op.Run()

	resp := <-respCh
	assert.ErrorIs(t, resp.err, performErr)
	assert.False(t, tx.commitCalled, "Commit should not have been called")
	assert.True(t, tx.rollbackCalled, "Rollback should have been called")
	mockStorage.AssertExpectations(t)
	mockAction.AssertExpectations(t)
}

func TestOperator_processItem_CommitFails(t *testing.T) {
	ctx := context.Background()
	commitErr := errors.New("commit failed")
	mockStorage := &MockIStorage{}
	mockAction := &actions.MockIAction{}

	tx := &mockTx{commitErr: commitErr}
	wt := storage.NewWriterForTestWithTx(tx)

	mockStorage.EXPECT().
		Write(mock.Anything).
		Return(wt, nil)
	mockAction.EXPECT().
		Perform(mock.Anything, wt).
		Return(nil)

	queue := make(chan ActionItem, 1)
	op := NewOperator(mockStorage, queue)

	respCh := make(chan ActionItemResponse, 1)
	queue <- ActionItem{ctx: ctx, action: mockAction, response: respCh}
	close(queue)

	op.Run()

	resp := <-respCh
	assert.ErrorIs(t, resp.err, commitErr)
	assert.True(t, tx.commitCalled, "Commit should have been called")
	mockStorage.AssertExpectations(t)
	mockAction.AssertExpectations(t)
}

func TestOperator_Run_ExitsOnClosedChannel(t *testing.T) {
	mockStorage := &MockIStorage{}
	queue := make(chan ActionItem)
	op := NewOperator(mockStorage, queue)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		op.Run()
	}()

	close(queue)
	wg.Wait()
}
