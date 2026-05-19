package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// OperationRunner owns the operation map and its lock. It is the single place
// that creates, updates, and evicts Operation records.
// Callers submit work via Submit; the map is accessible via Get and ListForInstance.
type OperationRunner struct {
	mu         sync.RWMutex
	operations map[string]*Operation
	timeout    time.Duration
}

func newOperationRunner(timeout time.Duration) *OperationRunner {
	return &OperationRunner{
		operations: make(map[string]*Operation),
		timeout:    timeout,
	}
}

// Submit schedules fn in a goroutine, records the operation, and returns it immediately.
// The caller receives a *Operation with Status="running" that is updated in-place.
func (r *OperationRunner) Submit(operationType, instanceID string, fn func(context.Context) error) *Operation {
	op := &Operation{
		ID:         newOperationID(),
		Type:       operationType,
		InstanceID: instanceID,
		Status:     "running",
		Message:    fmt.Sprintf("%s scheduled", operationType),
		StartedAt:  time.Now().UTC(),
	}
	r.mu.Lock()
	r.operations[op.ID] = op
	r.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		completedAt := time.Now().UTC()
		if err := fn(ctx); err != nil {
			completedAt = time.Now().UTC()
			r.mu.Lock()
			if existing, ok := r.operations[op.ID]; ok {
				existing.Status = "failed"
				existing.Error = err.Error()
				existing.Message = fmt.Sprintf("%s failed", operationType)
				existing.CompletedAt = &completedAt
			}
			r.mu.Unlock()
			return
		}
		completedAt = time.Now().UTC()
		r.mu.Lock()
		if existing, ok := r.operations[op.ID]; ok {
			existing.Status = "succeeded"
			existing.Message = fmt.Sprintf("%s completed", operationType)
			existing.CompletedAt = &completedAt
		}
		r.mu.Unlock()
	}()

	return op
}

// Record registers an externally-created operation in the runner's map.
// Used by submitInstanceOperation which builds the op before goroutine launch.
func (r *OperationRunner) Record(op *Operation) {
	r.mu.Lock()
	r.operations[op.ID] = op
	r.mu.Unlock()
}

// Update applies fn to the operation with the given ID under the write lock.
func (r *OperationRunner) Update(id string, fn func(*Operation)) {
	r.mu.Lock()
	if op, ok := r.operations[id]; ok {
		fn(op)
	}
	r.mu.Unlock()
}

// Get returns the operation with the given ID.
func (r *OperationRunner) Get(id string) (*Operation, bool) {
	r.mu.RLock()
	op, ok := r.operations[id]
	r.mu.RUnlock()
	return op, ok
}

// ListForInstance returns all operations for the given instanceID (unordered).
func (r *OperationRunner) ListForInstance(instanceID string) []*Operation {
	r.mu.RLock()
	var ops []*Operation
	for _, op := range r.operations {
		if op.InstanceID == instanceID {
			ops = append(ops, op)
		}
	}
	r.mu.RUnlock()
	return ops
}

// Cleanup removes completed operations whose CompletedAt is before cutoff.
func (r *OperationRunner) Cleanup(cutoff time.Time) {
	r.mu.Lock()
	for id, op := range r.operations {
		if op.CompletedAt != nil && op.CompletedAt.Before(cutoff) {
			delete(r.operations, id)
		}
	}
	r.mu.Unlock()
}

func newOperationID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("op-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
