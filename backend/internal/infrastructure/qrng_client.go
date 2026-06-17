package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type QRNGClient struct {
	baseURL string
	client  *http.Client
}

func NewQRNGClient() *QRNGClient {
	baseURL := os.Getenv("QRNG_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8002"
	}
	return &QRNGClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type QuantumRandomResult struct {
	Value        int    `json:"value"`
	Bits         []int  `json:"bits"`
	NQubits      int    `json:"n_qubits"`
	CircuitDepth int    `json:"circuit_depth"`
	Purpose      string `json:"purpose"`
}

func (c *QRNGClient) GetRandom(low, high int, purpose string) (*QuantumRandomResult, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"low":     low,
		"high":    high,
		"purpose": purpose,
	})

	resp, err := c.client.Post(
		fmt.Sprintf("%s/quantum/random", c.baseURL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("QRNG service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("QRNG service returned status %d", resp.StatusCode)
	}

	var result QuantumRandomResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding QRNG response: %w", err)
	}
	return &result, nil
}
