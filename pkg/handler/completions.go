package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/dwarvesf/gerr"
	"github.com/gin-gonic/gin"
)

var (
	url    = "https://chatapi.akash.network/api/v1/chat/completions"
	method = "POST"
)

const defaultModel = "Meta-Llama-3-1-8B-Instruct-FP8"

type completionsRequest struct {
	Role           string `json:"role" binding:"required"`
	Content        string `json:"content" binding:"required"`
	ConversationID string `json:"conversation_id"`
	Model          string `json:"model"`
}

// Response structure for the completion request
type CompletionResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	Content          string `json:"content"`
	Role             string `json:"role"`
	ConversationID   string `json:"conversation_id"`
	ConversationName string `json:"conversation_name"`
	CreatedAt        int64  `json:"created_at"`
}

// Define the structs to match the JSON structure
type ChoiceMessage struct {
	Content      string  `json:"content"`
	Role         string  `json:"role"`
	ToolCalls    *string `json:"tool_calls"` // Nullable fields use pointers
	FunctionCall *string `json:"function_call"`
}

type Choice struct {
	FinishReason string        `json:"finish_reason"`
	Index        int           `json:"index"`
	Message      ChoiceMessage `json:"message"`
}

type Usage struct {
	CompletionTokens        int     `json:"completion_tokens"`
	PromptTokens            int     `json:"prompt_tokens"`
	TotalTokens             int     `json:"total_tokens"`
	CompletionTokensDetails *string `json:"completion_tokens_details"`
	PromptTokensDetails     *string `json:"prompt_tokens_details"`
}

type AkashCompletionResponse struct {
	ID                string   `json:"id"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Object            string   `json:"object"`
	SystemFingerprint *string  `json:"system_fingerprint"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	ServiceTier       *string  `json:"service_tier"`
	PromptLogprobs    *string  `json:"prompt_logprobs"`
}

// handleError is a generic function that returns a value of type T and an error
func handleError[T any](msg string, err error) (T, error) {
	var zeroValue T
	fmt.Println(msg, err)
	return zeroValue, fmt.Errorf("%s: %w", msg, err)
}

// Completions godoc
// @Summary Send chat message
// @Description Send a chat message to the completions API and receive a response.
// @Tags chat
// @Accept json
// @Produce json
// @Param request body completionsRequest true "Chat message request body"
// @Success 200 {object} CompletionResponse
// @Failure 400 {object} ErrorResponse "Bad Request"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Router /send-chat [post]
func (h *Handler) Completions(c *gin.Context) {
	var req completionsRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Get the user ID from the header
	userID := c.Request.Header.Get("user-id")

	// If the conversation ID is not provided, get the most recent conversation ID and increment it by 1
	if req.ConversationID == "" {
		id, err := getMostRecentConversationID(h.db)
		if err != nil {
			h.handleError(c, gerr.E(500, gerr.Trace(err)))
			return
		}
		// Increment the conversation ID by 1
		req.ConversationID = strconv.Itoa(id + 1)
	}

	// If the model is not provided, use the default model
	if req.Model == "" {
		req.Model = defaultModel
	}

	res, err := h.doCompletions(req, userID, req.ConversationID, req.Model)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doCompletions(cReq completionsRequest, userID, conversationID, model string) (CompletionResponse, error) {
	_, err := ensureConversation(h.db, conversationID, model, userID)
	if err != nil {
		return handleError[CompletionResponse]("Error ensuring conversation:", err)
	}

	oldMsgs, err := getOldMessages(h.db, conversationID)
	if err != nil {
		return handleError[CompletionResponse]("Error getting old messages:", err)
	}

	oldMsgs = append(oldMsgs, map[string]string{
		"role":    cReq.Role,
		"content": cReq.Content,
	})
	err = h.setMessages(conversationID, ChoiceMessage{
		Role:    cReq.Role,
		Content: cReq.Content,
	}, time.Now().Unix())
	if err != nil {
		return handleError[CompletionResponse]("Error inserting message into DB:", err)
	}

	payload := map[string]interface{}{
		"model":    model,
		"messages": oldMsgs,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return handleError[CompletionResponse]("Error marshalling JSON:", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return handleError[CompletionResponse]("Error creating request:", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer sk-rMHU18txLCr23JJQNoR9hw")

	res, err := client.Do(req)
	if err != nil {
		return handleError[CompletionResponse]("Error making API request:", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return handleError[CompletionResponse]("Error reading response body:", err)
	}

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		return handleError[CompletionResponse]("Error indenting JSON:", err)
	}

	var completionResponse AkashCompletionResponse
	err = json.Unmarshal(prettyJSON.Bytes(), &completionResponse)
	if err != nil {
		return handleError[CompletionResponse]("Error unmarshaling JSON:", err)
	}

	err = h.setMessages(conversationID, completionResponse.Choices[0].Message, completionResponse.Created)
	if err != nil {
		return handleError[CompletionResponse]("Error inserting completion message into DB:", err)
	}

	var summarizeResponse AkashCompletionResponse
	if len(oldMsgs) == 3 {
		msgs := append(oldMsgs, map[string]string{
			"role":    completionResponse.Choices[0].Message.Role,
			"content": completionResponse.Choices[0].Message.Content,
		},
			map[string]string{
				"role":    "user",
				"content": "Summarize this conversation in 4-5 concise words",
			},
		)

		payload := map[string]interface{}{
			"model":    "Meta-Llama-3-1-8B-Instruct-FP8",
			"messages": msgs,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return handleError[CompletionResponse]("Error marshalling JSON:", err)
		}

		client := &http.Client{}
		req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
		if err != nil {
			return handleError[CompletionResponse]("Error creating request:", err)
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer sk-rMHU18txLCr23JJQNoR9hw")

		res, err := client.Do(req)
		if err != nil {
			return handleError[CompletionResponse]("Error making API request:", err)
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return handleError[CompletionResponse]("Error reading response body:", err)
		}

		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, body, "", "  ")
		if err != nil {
			return handleError[CompletionResponse]("Error indenting JSON:", err)
		}

		err = json.Unmarshal(prettyJSON.Bytes(), &summarizeResponse)
		if err != nil {
			return handleError[CompletionResponse]("Error unmarshaling JSON:", err)
		}

		_, err = insertName(h.db, conversationID, summarizeResponse.Choices[0].Message.Content)
		if err != nil {
			return handleError[CompletionResponse]("Error insert name:", err)
		}

		return CompletionResponse{
			Success:          true,
			Message:          "Successfully completed the request.",
			Content:          completionResponse.Choices[0].Message.Content,
			Role:             completionResponse.Choices[0].Message.Role,
			ConversationID:   conversationID,
			CreatedAt:        completionResponse.Created,
			ConversationName: summarizeResponse.Choices[0].Message.Content,
		}, nil
	}

	// Prepare the final response structure
	response := CompletionResponse{
		Success:        true,
		Message:        "Successfully completed the request.",
		Content:        completionResponse.Choices[0].Message.Content,
		Role:           completionResponse.Choices[0].Message.Role,
		ConversationID: conversationID,
		CreatedAt:      completionResponse.Created,
	}

	return response, nil
}

func getOldMessages(db *sql.DB, conversationID string) ([]map[string]string, error) {
	var messages []map[string]string

	rows, err := db.Query(`
		SELECT role, content FROM messages WHERE conversation_id = $1 ORDER BY timestamp ASC`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch old messages: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, fmt.Errorf("could not scan message: %v", err)
		}
		// Add the old messages to the slice
		messages = append(messages, map[string]string{
			"role":    role,
			"content": content,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate over rows: %v", err)
	}

	return messages, nil
}

func (h *Handler) setMessages(conversationID string, message ChoiceMessage, created int64) error {
	// Insert the new message into the messages table
	_, err := h.db.Exec(`
		INSERT INTO messages (conversation_id, role, content, timestamp)
		VALUES ($1, $2, $3, $4)`,
		conversationID, message.Role, message.Content, created)
	if err != nil {
		return fmt.Errorf("could not insert message: %v", err)
	}

	return nil
}

// ensureConversation ensures that a conversation exists with the provided conversationID.
// If not, it creates a new conversation.
func ensureConversation(db *sql.DB, conversationID string, model, userID string) (string, error) {
	// Check if the conversation already exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM conversations WHERE id = $1)`, conversationID).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("could not check if conversation exists: %v", err)
	}

	// If conversation exists, return the existing conversationID
	if exists {
		return conversationID, nil
	}

	// If conversation doesn't exist, create a new one
	var newConversationID string
	err = db.QueryRow(`
		INSERT INTO conversations (id, model, conversation_name ,user_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(id) DO NOTHING
		RETURNING id`, conversationID, model, "New Conversation", userID, time.Now().Unix()).Scan(&newConversationID)
	if err != nil {
		return "", fmt.Errorf("could not create conversation: %v", err)
	}

	// Return the newly created conversationID
	if newConversationID == "" {
		// If no new conversation was created, use the original conversationID
		return conversationID, nil
	}

	return newConversationID, nil
}

func insertName(db *sql.DB, conversationID string, conversation_name string) (string, error) {
	var existingID string

	// Check if the conversation already exists
	err := db.QueryRow(`SELECT id FROM conversations WHERE id = $1`, conversationID).Scan(&existingID)
	if err != nil {
		if err == sql.ErrNoRows {
			// If not found, insert a new row
			err = db.QueryRow(
				`INSERT INTO conversations (id, conversation_name) 
                 VALUES ($1, $2) 
                 RETURNING id`,
				conversationID, conversation_name,
			).Scan(&existingID)
			if err != nil {
				return "", fmt.Errorf("could not insert conversation: %v", err)
			}
			return existingID, nil
		}
		// Return other errors during the SELECT operation
		return "", fmt.Errorf("could not check conversation existence: %v", err)
	}

	// If found, update the conversation name
	_, err = db.Exec(`UPDATE conversations SET conversation_name = $1 WHERE id = $2`, conversation_name, conversationID)
	if err != nil {
		return "", fmt.Errorf("could not update conversation name: %v", err)
	}

	return existingID, nil
}

func getMostRecentConversationID(db *sql.DB) (int, error) {
	var conversationID int
	err := db.QueryRow("SELECT id FROM conversations ORDER BY id DESC LIMIT 1").Scan(&conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No conversations found
		}
		return 0, fmt.Errorf("could not fetch the most recent conversation ID: %v", err)
	}
	return conversationID, nil
}
