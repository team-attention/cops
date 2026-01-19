package txmanager

import "context"

// TransactionFunc is a function that executes within a transaction.
// The ctx parameter contains the transaction context and must be passed to all repository operations.
// If the function returns an error, the transaction is rolled back.
// If the function returns nil, the transaction is committed.
type TransactionFunc func(ctx context.Context) (interface{}, error)

// TransactionManagerPort defines the interface for managing database transactions.
// This abstraction allows the service layer to use transactions without depending on
// database-specific types (e.g., mongo.Client, mongo.SessionContext).
type TransactionManagerPort interface {
	// WithTransaction executes a function within a transaction.
	// It automatically handles session creation, commit, rollback, and cleanup.
	//
	// Parameters:
	//   - ctx: Parent context for timeout/cancellation
	//   - fn: Function to execute within transaction
	//
	// Returns:
	//   - result: Value returned by fn if transaction commits successfully
	//   - error: Error from fn (triggers rollback) or transaction infrastructure error
	//
	// Behavior:
	//   - If fn returns (result, nil): Transaction commits, returns (result, nil)
	//   - If fn returns (nil, error): Transaction rolls back, returns (nil, error)
	//   - Automatically retries on transient errors (network issues, etc.)
	WithTransaction(ctx context.Context, fn TransactionFunc) (interface{}, error)
}
