package jsonl

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
	"github.com/team-attention/cops/shared/domain"
)

// JSONLParser implements ParserPort for Claude Code JSONL files.
type JSONLParser struct {
	logger *slog.Logger
}

// NewJSONLParser creates a new JSONL parser.
func NewJSONLParser(l *slog.Logger) *JSONLParser {
	return &JSONLParser{
		logger: l.With(slog.String("name", "tracking.parser.jsonl")),
	}
}

// ParseSessionFiles parses all JSONL files in a project's Claude directory.
func (p *JSONLParser) ParseSessionFiles(claudeProjectDir string) ([]*domain.Record, error) {
	var records []*domain.Record

	// Check if directory exists
	if _, err := os.Stat(claudeProjectDir); os.IsNotExist(err) {
		p.logger.Debug("claude project directory does not exist",
			slog.String("path", claudeProjectDir))
		return records, nil
	}

	files, err := filepath.Glob(filepath.Join(claudeProjectDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	p.logger.Debug("found JSONL files",
		slog.Int("count", len(files)),
		slog.String("path", claudeProjectDir))

	for _, file := range files {
		fileRecords, err := p.parseFile(file)
		if err != nil {
			p.logger.Warn("failed to parse file",
				slog.String("file", file),
				slog.Any("error", err))
			continue
		}
		records = append(records, fileRecords...)
	}

	return records, nil
}

func (p *JSONLParser) parseFile(filePath string) ([]*domain.Record, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []*domain.Record
	scanner := bufio.NewScanner(file)
	// Increase buffer for large lines (Claude messages can be very long)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record domain.Record
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			p.logger.Debug("skipping malformed line",
				slog.String("file", filePath),
				slog.Int("line", lineNum),
				slog.Any("error", err))
			continue
		}

		// Parse ALL record types (user, assistant, file-history-snapshot)
		// The Record.UnmarshalJSON handles type dispatch automatically
		records = append(records, &record)
	}

	if err := scanner.Err(); err != nil {
		return records, err
	}

	return records, nil
}

// Compile-time interface verification
var _ parser.ParserPort = (*JSONLParser)(nil)
