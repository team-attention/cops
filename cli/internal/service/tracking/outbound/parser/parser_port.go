package parser

import "github.com/team-attention/cops/shared/domain"

// ParserPort defines the interface for parsing session files.
type ParserPort interface {
	// ParseSessionFiles parses all JSONL files in a project's Claude directory.
	ParseSessionFiles(claudeProjectDir string) ([]*domain.SessionRecord, error)
}
