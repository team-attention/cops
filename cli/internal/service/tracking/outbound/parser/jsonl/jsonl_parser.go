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
func (p *JSONLParser) ParseSessionFiles(claudeProjectDir string) ([]*domain.Transcript, error) {
	var transcripts []*domain.Transcript

	// Check if directory exists
	if _, err := os.Stat(claudeProjectDir); os.IsNotExist(err) {
		p.logger.Debug("claude project directory does not exist",
			slog.String("path", claudeProjectDir))
		return transcripts, nil
	}

	files, err := filepath.Glob(filepath.Join(claudeProjectDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	p.logger.Debug("found JSONL files",
		slog.Int("count", len(files)),
		slog.String("path", claudeProjectDir))

	for _, file := range files {
		fileTranscripts, err := p.parseFile(file)
		if err != nil {
			p.logger.Warn("failed to parse file",
				slog.String("file", file),
				slog.Any("error", err))
			continue
		}
		transcripts = append(transcripts, fileTranscripts...)
	}

	return transcripts, nil
}

func (p *JSONLParser) parseFile(filePath string) ([]*domain.Transcript, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var transcripts []*domain.Transcript
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

		var transcript domain.Transcript
		if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
			p.logger.Debug("skipping malformed line",
				slog.String("file", filePath),
				slog.Int("line", lineNum),
				slog.Any("error", err))
			continue
		}

		// Parse ALL transcript types (user, assistant, system, summary, file-history-snapshot)
		// The Transcript.UnmarshalJSON handles type dispatch automatically
		transcripts = append(transcripts, &transcript)
	}

	if err := scanner.Err(); err != nil {
		return transcripts, err
	}

	return transcripts, nil
}

// Compile-time interface verification
var _ parser.ParserPort = (*JSONLParser)(nil)
