package domain

// ContentBlockType represents the type discriminator for content blocks.
type ContentBlockType string

const (
	ContentBlockTypeText       ContentBlockType = "text"
	ContentBlockTypeToolUse    ContentBlockType = "tool_use"
	ContentBlockTypeToolResult ContentBlockType = "tool_result"
)

// ContentBlock is the interface implemented by all content block types.
type ContentBlock interface {
	BlockType() ContentBlockType
}

// TextContentBlock represents a text content block.
type TextContentBlock struct {
	Type ContentBlockType `json:"type"`
	Text string           `json:"text"`
}

// BlockType implements ContentBlock interface.
func (b *TextContentBlock) BlockType() ContentBlockType { return ContentBlockTypeText }

// ToolUseContentBlock represents a tool use content block.
type ToolUseContentBlock struct {
	Type  ContentBlockType `json:"type"`
	ID    string           `json:"id"`
	Name  string           `json:"name"`
	Input map[string]any   `json:"input"`
}

// BlockType implements ContentBlock interface.
func (b *ToolUseContentBlock) BlockType() ContentBlockType { return ContentBlockTypeToolUse }

// ToolResultContentBlock represents a tool result content block.
type ToolResultContentBlock struct {
	Type      ContentBlockType `json:"type"`
	ToolUseID string           `json:"tool_use_id"`
	Content   string           `json:"content"`
	IsError   bool             `json:"is_error,omitempty"`
}

// BlockType implements ContentBlock interface.
func (b *ToolResultContentBlock) BlockType() ContentBlockType { return ContentBlockTypeToolResult }
