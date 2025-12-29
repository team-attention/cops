package domain

type RecordType string

const (
	RecordTypeFileHistorySnapshot RecordType = "file-history-snapshot"
	RecordTypeUser                RecordType = "user"
	RecordTypeMessage             RecordType = "message"
)

type Record struct {
	Type RecordType
}

type RecordBase struct {
	Type RecordType
}
