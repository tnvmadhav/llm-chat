package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var openAIKey = os.Getenv("OPENAI_API_KEY")

const apiURL = "https://api.openai.com/v1/chat/completions"
const LLM = "gpt-3.5-turbo"

// OpenAIResponse represents the structure of the OpenAI API response
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens   int `json:"prompt_tokens"`
		CompletionUsed int `json:"completion_used"`
		Turns          int `json:"turns"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role     string `json:"role"`
			Content  string `json:"content"`
			Children []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"children"`
		} `json:"message"`
	} `json:"choices"`
}

func getOpenAIResponse(prompt []map[string]string) (string, error) {
	payload := map[string]interface{}{
		"model": LLM,
		"messages": append(
			[]map[string]string{
				{"role": "system", "content": "You're a helpful AI assisstant. Provide answers in correct markdown syntax please."},
			},
			prompt...,
		),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling JSON payload: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openAIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var openAIResp OpenAIResponse
	err = json.Unmarshal(body, &openAIResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling JSON response: %v", err)
	}

	// Extract the assistant's reply from the OpenAI response
	var assistantReply string
	if len(openAIResp.Choices) > 0 {
		assistantReply = openAIResp.Choices[0].Message.Content
	}

	return assistantReply, nil
}

func GetOpenAIMessageStr(conversation []map[string]string) string {
	response, err := getOpenAIResponse(conversation)
	if err != nil {
		fmt.Println("Error:", err)
		return "LLM Error :("
	}
	return response
}
