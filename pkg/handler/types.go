package handler

// Conversation struct represents a conversation with an array of messages
type Conversation struct {
	ID               string `gorm:"primaryKey;autoIncrement"` // Unique ID for the conversation
	Model            string `json:"model"`                    // Model used for the conversation
	ConversationName string `json:"conversationName"`         // Name of the conversation
	UserID           string `json:"userID"`                   // User ID associated with the conversation
	CreatedAt        string `json:"createdAt"`                // Time the conversation was created
}

type Message struct {
	ID             int    `json:"id"`
	ConversationID int    `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	Timestamp      string `json:"timestamp"`
}

// ErrorResponse represents the structure for error messages
// @Description Common error response format
// @Success 500 {object} ErrorResponse "Internal Server Error"
// @Failure 400 {object} ErrorResponse "Bad Request"
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
}
