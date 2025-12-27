package domain

import (
	"encoding/json"
	"fmt"
)

// MessageContent wraps the polymorphic content field.
// Content can be either a string (user messages) or []ContentBlock (assistant messages).
type MessageContent struct {
	Text     *string        // When content is string (nil when IsBlocks=true)
	Blocks   []ContentBlock // When content is []ContentBlock (nil when IsBlocks=false)
	IsBlocks bool           // Internal discriminator
}

// UnmarshalJSON implements custom unmarshaling for polymorphic content.
func (c *MessageContent) UnmarshalJSON(data []byte) error {
	// Try string first
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		c.Text = &text
		c.IsBlocks = false
		return nil
	}

	// Otherwise, try to unmarshal as an array of content blocks
	var rawBlocks []json.RawMessage
	if err := json.Unmarshal(data, &rawBlocks); err != nil {
		return fmt.Errorf("content must be string or array: %w", err)
	}

	c.Blocks = make([]ContentBlock, 0, len(rawBlocks))
	c.IsBlocks = true

	for i, raw := range rawBlocks {
		// First extract the type field
		var typeHolder struct {
			Type ContentBlockType `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeHolder); err != nil {
			return fmt.Errorf("failed to parse content block %d type: %w", i, err)
		}

		// Unmarshal into the correct concrete type
		var block ContentBlock
		switch typeHolder.Type {
		case ContentBlockTypeText:
			var tb TextContentBlock
			if err := json.Unmarshal(raw, &tb); err != nil {
				return fmt.Errorf("failed to parse text block %d: %w", i, err)
			}
			block = &tb
		case ContentBlockTypeToolUse:
			var tb ToolUseContentBlock
			if err := json.Unmarshal(raw, &tb); err != nil {
				return fmt.Errorf("failed to parse tool_use block %d: %w", i, err)
			}
			block = &tb
		case ContentBlockTypeToolResult:
			var tb ToolResultContentBlock
			if err := json.Unmarshal(raw, &tb); err != nil {
				return fmt.Errorf("failed to parse tool_result block %d: %w", i, err)
			}
			block = &tb
		case ContentBlockTypeThinking:
			var tb ThinkingContentBlock
			if err := json.Unmarshal(raw, &tb); err != nil {
				return fmt.Errorf("failed to parse thinking block %d: %w", i, err)
			}
			block = &tb
		default:
			// Unknown type - skip for forward compatibility
			continue
		}
		c.Blocks = append(c.Blocks, block)
	}

	return nil
}

// MarshalJSON implements custom marshaling for polymorphic content.
func (c MessageContent) MarshalJSON() ([]byte, error) {
	if c.IsBlocks {
		if c.Blocks == nil {
			return json.Marshal([]ContentBlock{})
		}
		return json.Marshal(c.Blocks)
	}
	if c.Text != nil {
		return json.Marshal(*c.Text)
	}
	// Return null instead of empty string for uninitialized content
	return []byte("null"), nil
}

// Message contains the role and content of a session message.
type Message struct {
	ID           string          `json:"id,omitempty"`
	Type         string          `json:"type,omitempty"`
	Role         string          `json:"role"`
	Model        string          `json:"model,omitempty"`
	Content      *MessageContent `json:"content,omitempty"`
	StopReason   string          `json:"stop_reason,omitempty"`
	StopSequence string          `json:"stop_sequence,omitempty"`
	Usage        *Usage          `json:"usage,omitempty"`
}
