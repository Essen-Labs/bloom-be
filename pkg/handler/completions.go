package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dwarvesf/gerr"
	"github.com/gin-gonic/gin"
)

var (
	url    = "https://chatapi.akash.network/api/v1/chat/completions"
	method = "POST"
)

type completionsRequest struct {
	Role    string `json:"role" binding:"required"`
	Content string `json:"content" binding:"required"`
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

	res, err := h.doCompletions(req, userID)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doCompletions(cReq completionsRequest, userID string) ([]byte, error) {
	payload := map[string]interface{}{
		"model": "Meta-Llama-3-1-8B-Instruct-FP8",
		"messages": []map[string]string{
			{
				"role":    cReq.Role,
				"content": cReq.Content,
			},
		},
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

	return prettyJSON.Bytes(), nil
}
