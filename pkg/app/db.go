package app

import (
	"log"
)

// Message struct represents a message
type Message struct {
	Role      string `json:"role"`      // Role of the sender (e.g., 'user' or 'assistant')
	Content   string `json:"content"`   // Message content
	Timestamp string `json:"timestamp"` // Timestamp of when the message was sent
}

// Conversation struct represents a conversation with an array of messages
type Conversation struct {
	ID       int       `gorm:"primaryKey;autoIncrement"`                  // Unique ID for the conversation
	Model    string    `json:"model"`                                     // Model used for the conversation
	UserID   string    `json:"userID"`                                    // User ID associated with the conversation
	Messages []Message `json:"messages" gorm:"foreignKey:ConversationID"` // Messages in the conversation
}

func (a App) createTables() error {
	// Create Conversations table
	_, err := a.db.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			id SERIAL PRIMARY KEY,
			model VARCHAR(255),
			user_id VARCHAR(255)
		)
	`)
	if err != nil {
		log.Fatal("Error creating conversations table: ", err)
		return err
	}

	// Create Messages table
	_, err = a.db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			conversation_id INTEGER REFERENCES conversations(id) ON DELETE CASCADE,
			role VARCHAR(255),
			content TEXT,
			timestamp VARCHAR(255)
		)
	`)
	if err != nil {
		log.Fatal("Error creating messages table: ", err)
		return err
	}
	return nil
}
