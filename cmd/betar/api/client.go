package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BaseURL string
	Client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

func (c *Client) Get(path string, result interface{}) error {
	resp, err := c.Client.Get(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) Post(path string, body interface{}, result interface{}) error {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	resp, err := c.Client.Post(c.BaseURL+path, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// PaymentRequiredError is returned when a 402 Payment Required is received
type PaymentRequiredError struct {
	AgentID            string      `json:"agent_id"`
	RequestID          string      `json:"request_id"`
	Message            string      `json:"message"`
	PaymentRequirement interface{} `json:"payment_requirement"`
	RequiresPayment    bool        `json:"requires_payment"`
}

// PostWithPayment handles 402 Payment Required responses
func (c *Client) PostWithPayment(path string, body interface{}, result interface{}) (*PaymentRequiredError, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	resp, err := c.Client.Post(c.BaseURL+path, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Check for 402 Payment Required
	if resp.StatusCode == http.StatusPaymentRequired {
		var payErr PaymentRequiredError
		if err := json.Unmarshal(respBody, &payErr); err != nil {
			return nil, fmt.Errorf("failed to decode 402 response: %w", err)
		}
		return &payErr, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil, nil
}
