package domain

// ID represents a unique identifier for domain entities.
// All entity identifiers in the application must use this type.
type ID string

// String returns the string representation of the ID.
func (id ID) String() string {
	return string(id)
}

// IsEmpty checks if the ID is empty.
func (id ID) IsEmpty() bool {
	return id == ""
}

// NewID creates a new ID from a string value.
func NewID(s string) ID {
	return ID(s)
}
