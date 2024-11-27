package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dwarvesf/gerr"
	"github.com/gin-gonic/gin"
)

// GetChatByIDRequest represents the request structure to get a chat by ID
// @Description Request to get a specific conversation by its ID
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} GetChatByIDResponse "Successfully retrieved the conversation"
// @Failure 400 {object} ErrorResponse "Invalid input data"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chat/{conversation_id} [get]
type GetChatByIDRequest struct {
	ConversationID string `json:"conversation_id"`
}

// GetChatByIDResponse represents the response structure for getting a conversation by ID
// @Description Response when a conversation is successfully retrieved
// @Success 200 {object} GetChatByIDResponse "Successfully retrieved the conversation"
// @Failure 404 {object} ErrorResponse "Conversation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chat/{conversation_id} [get]
type GetChatByIDResponse struct {
	Success      bool         `json:"success"`
	Message      string       `json:"message"`
	Conversation Conversation `json:"conversation"`
}

// GetAllChatResponse represents the response structure for getting all conversations
// @Description Response containing all conversations for a user
// @Success 200 {object} GetAllChatResponse "Successfully retrieved all conversations"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chat/all [get]
type GetAllChatResponse struct {
	Success       bool           `json:"success"`
	Message       string         `json:"message"`
	Conversations []Conversation `json:"conversations"`
}

// GetChatById gets a conversation by its ID
// @Summary Get a conversation by ID
// @Description Retrieves a conversation from the database using its ID
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} GetChatByIDResponse "Successfully retrieved the conversation"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "Conversation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chat/{conversation_id} [get]
func (h *Handler) GetChatById(c *gin.Context) {
	var req GetChatByIDRequest

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

func (h *Handler) doGetChatByID(cReq GetChatByIDRequest) ([]byte, error) {
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

	response := GetChatByIDResponse{
		Success:      true,
		Message:      fmt.Sprintf("Conversation with ID %s found", cReq.ConversationID),
		Conversation: conversation,
	}
	return json.Marshal(response)
}

// GetAllChat retrieves all conversations
// @Summary Get all conversations
// @Description Retrieves all conversations stored in the database
// @Success 200 {object} GetAllChatResponse "Successfully retrieved all conversations"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chat/all [get]
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

	response := GetAllChatResponse{
		Success:       true,
		Message:       "Conversations found",
		Conversations: conversations,
	}

	return json.Marshal(response)
}


type deleteChatByIDRequest struct {
	ConversationID string `json:"conversation_id"`
}

type deleteChatByIDResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DeleteChatById deletes a conversation by its ID
// @Summary Delete a conversation by ID
// @Description Deletes the specified conversation and its associated messages from the database
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} deleteChatByIDResponse "Successfully deleted the conversation"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "Conversation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chat/{conversation_id} [delete]
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

// DeleteAllChat deletes all conversations for a user
// @Summary Delete all conversations for a user
// @Description Deletes all conversations and their associated messages for a given user
// @Security BearerAuth
// @Success 200 {object} deleteAllChatByUserIDResponse "Successfully deleted all conversations for the user"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "No conversations found for the user"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chat/all [delete]
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
