package app

import (
	"log"
)

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
