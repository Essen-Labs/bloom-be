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

// Define the structs to match the JSON structure
type Message struct {
	Content      string  `json:"content"`
	Role         string  `json:"role"`
	ToolCalls    *string `json:"tool_calls"` // Nullable fields use pointers
	FunctionCall *string `json:"function_call"`
}

type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
	Message      Message `json:"message"`
}

type Usage struct {
	CompletionTokens        int     `json:"completion_tokens"`
	PromptTokens            int     `json:"prompt_tokens"`
	TotalTokens             int     `json:"total_tokens"`
	CompletionTokensDetails *string `json:"completion_tokens_details"`
	PromptTokensDetails     *string `json:"prompt_tokens_details"`
}

type CompletionResponse struct {
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

// Signup create a user
func (h *Handler) Completions(c *gin.Context) {
	var req completionsRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

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

func (h *Handler) doCompletions(cReq completionsRequest, userID, conversationID, model string) ([]byte, error) {
	ensureConversation(h.db, conversationID, model, userID)
	oldMsgs, err := getOldMessages(h.db, conversationID)
	if err != nil {
		fmt.Println("Error getting old messages:", err)
		return nil, err
	}
	oldMsgs = append(oldMsgs, map[string]string{
		"role":    cReq.Role,
		"content": cReq.Content,
	})
	err = h.setMessages(conversationID, Message{
		Role:    cReq.Role,
		Content: cReq.Content,
	}, time.Now().Unix())
	if err != nil {
		fmt.Println("Error insert db:", err)
		return nil, err
	}

	payload := map[string]interface{}{
		"model":    "Meta-Llama-3-1-8B-Instruct-FP8",
		"messages": oldMsgs,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer sk-rMHU18txLCr23JJQNoR9hw")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		fmt.Println("Error indenting JSON:", err)
		return nil, err
	}

	var completionResponse CompletionResponse
	err = json.Unmarshal(prettyJSON.Bytes(), &completionResponse)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return nil, err
	}

	err = h.setMessages(conversationID, completionResponse.Choices[0].Message, completionResponse.Created)
	if err != nil {
		fmt.Println("Error insert db:", err)
		return nil, err
	}
	return prettyJSON.Bytes(), nil
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

func (h *Handler) setMessages(conversationID string, message Message, created int64) error {
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
		INSERT INTO conversations (id, model, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT(id) DO NOTHING
		RETURNING id`, conversationID, model, userID).Scan(&newConversationID)
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
