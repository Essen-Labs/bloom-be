package handler

// Conversation struct represents a conversation with an array of messages
type Conversation struct {
	ID     string `gorm:"primaryKey;autoIncrement"` // Unique ID for the conversation
	Model  string `json:"model"`                    // Model used for the conversation
	UserID string `json:"userID"`                   // User ID associated with the conversation
}

type Message struct {
	ID             int    `json:"id"`
	ConversationID int    `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	Timestamp      string `json:"timestamp"`
}