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

// Signup create a user
func (h *Handler) Completions(c *gin.Context) {
	var req completionsRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	res, err := h.doCompletions(req)
	if err != nil {
		h.handleError(c, gerr.E(500, gerr.Trace(err)))
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) doCompletions(cReq completionsRequest) ([]byte, error) {
	jsonData, err := json.Marshal(cReq)
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

	return prettyJSON.Bytes(), nil
}
