package v2

// HumanContentBlockType represents the type of content block in a human message.
type HumanContentBlockType string

const (
	HumanContentBlockTypeText  HumanContentBlockType = "text"
	HumanContentBlockTypeImage HumanContentBlockType = "image"
)

// HumanMessage represents user input in the conversation tree.
// Contains only text and image content (no tool_result - those are in ToolExecution).
type HumanMessage struct {
	TreeNodeMeta `bson:",inline"`

	Content []*HumanContentBlock `json:"content" bson:"content"`
	IsMeta  bool                 `json:"isMeta,omitempty" bson:"isMeta,omitempty"`
	Todos   []*Todo              `json:"todos,omitempty" bson:"todos,omitempty"`
}

// HumanContentBlock represents a content block in a human message.
// Supports text and image types only (tool_result is separated into ToolExecution).
type HumanContentBlock struct {
	Type HumanContentBlockType `json:"type" bson:"type"`

	// For type="text"
	Text *string `json:"text,omitempty" bson:"text,omitempty"`

	// For type="image"
	Image *ImageData `json:"image,omitempty" bson:"image,omitempty"`
}
