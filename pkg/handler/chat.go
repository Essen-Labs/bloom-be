package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
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
// @Param conversation_id is path string true "Conversation ID"
// @Success 200 {object} GetChatByIDResponse "Successfully retrieved the conversation"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "Conversation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /get-chat-by-id/{conversation_id} [get]
func (h *Handler) GetChatById(c *gin.Context) {
	conversationID := c.Param("conversation_id")

	res, err := h.doGetChatByID(conversationID)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doGetChatByID(conversationID string) ([]byte, error) {
	var conversation Conversation

	err := h.db.QueryRow(`
        SELECT id, model, conversation_name, user_id FROM conversations WHERE id = $1`, conversationID).Scan(
		&conversation.ID, &conversation.Model, &conversation.ConversationName, &conversation.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, fmt.Errorf("could not fetch conversation: %v", err)
	}

	response := GetChatByIDResponse{
		Success:      true,
		Message:      fmt.Sprintf("Conversation with ID %s found", conversationID),
		Conversation: conversation,
	}
	return json.Marshal(response)
}

// GetAllChat retrieves all conversations
// @Summary Get all conversations
// @Description Retrieves all conversations stored in the database
// @Success 200 {object} GetAllChatResponse "Successfully retrieved all conversations"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /get-chat-list [get]
func (h *Handler) GetAllChat(c *gin.Context) {
	// Get the user ID from the header
	userID := c.Request.Header.Get("user-id")

	res, err := h.doGetAllChat(userID)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doGetAllChat(userID string) ([]byte, error) {
	// Query to get all conversations
	rows, err := h.db.Query("SELECT id, model, conversation_name, user_id FROM conversations WHERE user_id = $1", userID)
	if err != nil {
		return nil, fmt.Errorf("error querying conversations: %v", err)
	}
	defer rows.Close()

	// Slice to hold the results
	var conversations []Conversation

	// Iterate over the rows
	for rows.Next() {
		var conversation Conversation
		if err := rows.Scan(&conversation.ID, &conversation.Model, &conversation.ConversationName, &conversation.UserID); err != nil {
			return nil, fmt.Errorf("error scanning conversation: %v", err)
		}
		conversations = append(conversations, conversation)
	}

	// Check for any errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over conversations: %v", err)
	}

	var response GetAllChatResponse
	if len(conversations) == 0 {
		response = GetAllChatResponse{
			Success:       false,
			Message:       "Conversations not found",
			Conversations: conversations,
		}
	} else {
		response = GetAllChatResponse{
			Success:       true,
			Message:       "Conversations found",
			Conversations: conversations,
		}
	}

	return json.Marshal(response)
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
// @Router /delete-chat/{conversation_id} [delete]
func (h *Handler) DeleteChatById(c *gin.Context) {
	conversationID := c.Param("conversation_id")

	res, err := h.doDeleteChatByID(conversationID)
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

func (h *Handler) doDeleteChatByID(conversationID string) ([]byte, error) {
	tx, err := h.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %v", err)
	}

	// Rollback the transaction if there's an error
	defer func() {
		if p := recover(); p != nil {
			err = tx.Rollback()
			panic(err) // Re-raise the panic
		} else if err != nil {
			err = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	_, err = tx.Exec("DELETE FROM messages WHERE conversation_id = $1", conversationID)
	if err != nil {
		return nil, fmt.Errorf("could not delete messages: %v", err)
	}

	result, err := tx.Exec("DELETE FROM conversations WHERE id = $1", conversationID)
	if err != nil {
		return nil, fmt.Errorf("could not delete conversation: %v", err)
	}

	// Check if a row was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("could not check rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("conversation with id %s not found", conversationID)
	}

	// Step 3: Create a success response
	response := deleteChatByIDResponse{
		Success: true,
		Message: fmt.Sprintf("Conversation with ID %s deleted successfully", conversationID),
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
// @Router /delete-all-chat [delete]
func (h *Handler) DeleteAllChat(c *gin.Context) {
	// Get the user ID from the cookie
	userID, err := h.GetUserFromCookie(c)
	if err != nil {
		if err == http.ErrNoCookie {
			userID = h.SetUserCookie(c)
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
			err = tx.Rollback()
			panic(err) // Re-raise the panic
		} else if err != nil {
			err = tx.Rollback()
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

type EditChatRequest struct {
	NewName        string `json:"role" binding:"new_name"`
	ConversationID string `json:"content" binding:"conversation_id"`
}

func (h *Handler) EditChat(c *gin.Context) {
	req := EditChatRequest{}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Get the user ID from the header
	userID := c.Request.Header.Get("user-id")

	res, err := h.doEditChat(userID, req)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doEditChat(userID string, req EditChatRequest) ([]byte, error) {
	// Validate input
	if req.NewName == "" || req.ConversationID == "" {
		return nil, errors.New("new_name and conversation_id are required")
	}

	// SQL query to update the conversation name
	query := `
		UPDATE conversations 
		SET conversation_name = $1
		WHERE id = $2 AND user_id = $3
		RETURNING conversation_name
		`
	var updatedName string

	// Execute the query
	err := h.db.QueryRow(query, req.NewName, req.ConversationID, userID).Scan(&updatedName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("conversation not found or user not authorized")
		}
		return nil, err
	}

	// Prepare the response
	response := map[string]string{
		"id":                req.ConversationID,
		"conversation_name": updatedName,
	}

	// Marshal the response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return responseJSON, nil
}
