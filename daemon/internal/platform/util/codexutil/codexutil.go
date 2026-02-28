package codexutil

import "encoding/json"

// ExtractCwd extracts the working directory from a Codex CLI JSONL line.
// Looks for the session_meta event type which contains payload.cwd.
// Returns empty string if the line is not a session_meta event or cwd is not found.
func ExtractCwd(line string) string {
	var data struct {
		Type        string `json:"type"`
		SessionMeta struct {
			Payload struct {
				Cwd string `json:"cwd"`
			} `json:"payload"`
		} `json:"session_meta"`
	}

	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return ""
	}

	if data.Type != "session_meta" {
		return ""
	}

	return data.SessionMeta.Payload.Cwd
}

// ExtractCwdFromLines scans a batch of lines for the first session_meta
// with a cwd field and returns it.
func ExtractCwdFromLines(lines []string) string {
	for _, line := range lines {
		if cwd := ExtractCwd(line); cwd != "" {
			return cwd
		}
	}
	return ""
}
