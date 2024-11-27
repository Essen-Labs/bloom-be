package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dwarvesf/gerr"
	"github.com/gin-gonic/gin"
)

type getAllMsgsByIDRequest struct {
	ConversationID string `json:"conversation_id"`
}

// GetAllMsgsByID fetches all messages by conversation ID
// @Summary Get all messages for a specific conversation
// @Description Fetches all messages associated with a given conversation ID, ordered by timestamp.
// @Accept json
// @Produce json
// @Param conversation_id body string true "Conversation ID"
// @Success 200 {array} Message "List of messages in the conversation"
// @Failure 400 {object} ErrorResponse "Invalid conversation ID"
// @Failure 404 {object} ErrorResponse "Conversation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /messages [get]
func (h *Handler) GetAllMsgsByID(c *gin.Context) {
	var req getAllMsgsByIDRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	res, err := h.doGetAllMsgsByID(req)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doGetAllMsgsByID(cReq getAllMsgsByIDRequest) ([]byte, error) {
	rows, err := h.db.Query(`
		SELECT id, conversation_id, role, content, timestamp 
		FROM messages 
		WHERE conversation_id = $1
		ORDER BY timestamp ASC
	`, cReq.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("could not query messages: %v", err)
	}
	defer rows.Close()

	var messages []Message

	// Loop through the rows and scan data into the Message struct
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.Role, &message.Content, &message.Timestamp); err != nil {
			return nil, fmt.Errorf("error scanning message row: %v", err)
		}
		messages = append(messages, message)
	}

	// Check if there were errors during the iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over message rows: %v", err)
	}

	// Convert the slice of messages to JSON
	response, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("error marshaling messages to JSON: %v", err)
	}

	return response, nil
}
