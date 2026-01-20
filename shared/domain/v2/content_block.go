package session

// ImageData represents image content with base64-encoded data.
type ImageData struct {
	MediaType string `json:"mediaType" bson:"mediaType"` // e.g., "image/png", "image/jpeg"
	Data      string `json:"data" bson:"data"`           // Base64-encoded image data
}

// Todo represents a task item tracked during agent execution.
type Todo struct {
	Content    string `json:"content" bson:"content"`
	Status     string `json:"status" bson:"status"` // "pending", "in_progress", "completed"
	ActiveForm string `json:"activeForm" bson:"activeForm"`
}
