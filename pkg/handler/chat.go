package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dwarvesf/gerr"
	"github.com/gin-gonic/gin"
)

type getChatByIDRequest struct {
	ConversationID string `json:"conversation_id"`
}

func (h *Handler) GetChatById(c *gin.Context) {
	var req getChatByIDRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	res, err := h.doGetChatByID(req)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doGetChatByID(cReq getChatByIDRequest) ([]byte, error) {
	var conversation Conversation

	err := h.db.QueryRow(`
        SELECT id, model, user_id FROM conversations WHERE id = $1`, cReq.ConversationID).Scan(
		&conversation.ID, &conversation.Model, &conversation.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, fmt.Errorf("could not fetch conversation: %v", err)
	}

	response, err := json.Marshal(conversation)
	if err != nil {
		return nil, fmt.Errorf("could not marshal conversation: %v", err)
	}

	return response, nil
}

func (h *Handler) GetAllChat(c *gin.Context) {
	res, err := h.doGetAllChat()
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doGetAllChat() ([]byte, error) {
	// Query to get all conversations
	rows, err := h.db.Query("SELECT id, model, user_id FROM conversations")
	if err != nil {
		return nil, fmt.Errorf("error querying conversations: %v", err)
	}
	defer rows.Close()

	// Slice to hold the results
	var conversations []Conversation

	// Iterate over the rows
	for rows.Next() {
		var conversation Conversation
		if err := rows.Scan(&conversation.ID, &conversation.Model, &conversation.UserID); err != nil {
			return nil, fmt.Errorf("error scanning conversation: %v", err)
		}
		conversations = append(conversations, conversation)
	}

	// Check for any errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over conversations: %v", err)
	}

	// Convert the conversations to JSON
	response, err := json.Marshal(conversations)
	if err != nil {
		return nil, fmt.Errorf("error marshaling conversations to JSON: %v", err)
	}

	return response, nil
}

type deleteChatByIDRequest struct {
	ConversationID string `json:"conversation_id"`
}

type deleteChatByIDResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *Handler) DeleteChatById(c *gin.Context) {
	var req deleteChatByIDRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	res, err := h.doDeleteChatByID(req)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

type deleteAllChatByUserIDResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *Handler) doDeleteChatByID(cReq deleteChatByIDRequest) ([]byte, error) {
	tx, err := h.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %v", err)
	}

	// Rollback the transaction if there's an error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-raise the panic
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	_, err = tx.Exec("DELETE FROM messages WHERE conversation_id = $1", cReq.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("could not delete messages: %v", err)
	}

	result, err := tx.Exec("DELETE FROM conversations WHERE id = $1", cReq.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("could not delete conversation: %v", err)
	}

	// Check if a row was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("could not check rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("conversation with id %s not found", cReq.ConversationID)
	}

	// Step 3: Create a success response
	response := deleteChatByIDResponse{
		Success: true,
		Message: fmt.Sprintf("Conversation with ID %s deleted successfully", cReq.ConversationID),
	}

	// Marshal the response to JSON
	return json.Marshal(response)
}

func (h *Handler) DeleteAllChat(c *gin.Context) {

	// Get the user ID from the cookie
	userID, err := h.GetUserFromCookie(c)
	if err != nil {
		if err == http.ErrNoCookie {
			h.SetUserCookie(c)
		} else {
			h.handleError(c, err)
			return
		}
	}

	res, err := h.doDeleteAllChatByUserID(userID)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doDeleteAllChatByUserID(userID string) ([]byte, error) {
	// Start a transaction to delete messages and conversations atomically
	tx, err := h.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %v", err)
	}

	// Rollback the transaction if there's an error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-raise the panic
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	_, err = tx.Exec(`
		DELETE FROM messages
		WHERE conversation_id IN (
			SELECT id FROM conversations WHERE user_id = $1
		)
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("could not delete messages for user_id %s: %v", userID, err)
	}

	result, err := tx.Exec("DELETE FROM conversations WHERE user_id = $1", userID)
	if err != nil {
		return nil, fmt.Errorf("could not delete conversations for user_id %s: %v", userID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("could not check rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("no conversations found for user_id %s", userID)
	}

	response := deleteAllChatByUserIDResponse{
		Success: true,
		Message: fmt.Sprintf("Deleted %d conversations and their messages for user ID %s", rowsAffected, userID),
	}

	return json.Marshal(response)
}
