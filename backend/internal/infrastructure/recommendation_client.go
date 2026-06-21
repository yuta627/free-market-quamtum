package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type RecommendationClient struct {
	baseURL string
	client  *http.Client
}

func NewRecommendationClient() *RecommendationClient {
	baseURL := os.Getenv("RECOMMENDATION_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8001"
	}
	return &RecommendationClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// 量子カーネル計算用（50件分の回路実行があるため長めのタイムアウト）
func newSlowClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

type RecommendedItem struct {
	ItemID      int64   `json:"item_id"`
	Score       float64 `json:"score"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	IsColdStart bool    `json:"is_cold_start"`
}

type recommendationsResponse struct {
	QueryItemID int64             `json:"query_item_id"`
	Results     []RecommendedItem `json:"results"`
}

type metaRecommendationRequest struct {
	Title     string  `json:"title"`
	Price     float64 `json:"price"`
	Condition string  `json:"condition"`
	K         int     `json:"k"`
}

type metaRecommendationResponse struct {
	Results []RecommendedItem `json:"results"`
}

// GetSimilarItems calls the Python (FastAPI + FAISS) recommendation service.
// It first tries a lookup by item_id; if not found, falls back to content-based
// inference using the product's title, price, and condition.
func (c *RecommendationClient) GetSimilarItems(itemID uint, k int, title string, price int, condition string) ([]RecommendedItem, error) {
	url := fmt.Sprintf("%s/recommendations/%d?k=%d", c.baseURL, itemID, k)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("calling recommendation service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var parsed recommendationsResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, fmt.Errorf("decoding recommendation response: %w", err)
		}
		return parsed.Results, nil
	}

	if resp.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("recommendation service returned status %d", resp.StatusCode)
	}

	// item_id が FAISS インデックスにない場合 → コンテンツベースでリアルタイム推論
	return c.getSimilarByMeta(title, float64(price), condition, k)
}

// GetQMLSimilarItems calls the QML (PQC-based) recommendation endpoint.
func (c *RecommendationClient) GetQMLSimilarItems(itemID uint, k int) ([]RecommendedItem, error) {
	url := fmt.Sprintf("%s/recommendations/qml/%d?k=%d", c.baseURL, itemID, k)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("calling QML recommendation service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusServiceUnavailable {
		return []RecommendedItem{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("QML service returned status %d", resp.StatusCode)
	}
	var parsed recommendationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding QML response: %w", err)
	}
	return parsed.Results, nil
}

// GetClassicalSimilarItems calls the classical (PCA + FAISS) recommendation endpoint.
func (c *RecommendationClient) GetClassicalSimilarItems(itemID uint, k int) ([]RecommendedItem, error) {
	url := fmt.Sprintf("%s/recommendations/classical/%d?k=%d", c.baseURL, itemID, k)
	resp, err := c.client.Get(url)
	if err != nil {
		return []RecommendedItem{}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []RecommendedItem{}, nil
	}
	var parsed recommendationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return []RecommendedItem{}, nil
	}
	return parsed.Results, nil
}

// GetQKernelSimilarItems calls the quantum kernel recommendation endpoint.
func (c *RecommendationClient) GetQKernelSimilarItems(itemID uint, k int) ([]RecommendedItem, error) {
	url := fmt.Sprintf("%s/recommendations/qkernel/%d?k=%d", c.baseURL, itemID, k)
	resp, err := newSlowClient().Get(url)
	if err != nil {
		return []RecommendedItem{}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []RecommendedItem{}, nil
	}
	var parsed recommendationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return []RecommendedItem{}, nil
	}
	return parsed.Results, nil
}

func (c *RecommendationClient) GetSimilarByMeta(title string, price float64, condition string, k int) ([]RecommendedItem, error) {
	return c.getSimilarByMeta(title, price, condition, k)
}

func (c *RecommendationClient) getSimilarByMeta(title string, price float64, condition string, k int) ([]RecommendedItem, error) {
	body, _ := json.Marshal(metaRecommendationRequest{
		Title:     title,
		Price:     price,
		Condition: condition,
		K:         k,
	})

	resp, err := c.client.Post(
		fmt.Sprintf("%s/recommendations/by-meta", c.baseURL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return []RecommendedItem{}, nil // ML サーバー未起動時は空で返す
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []RecommendedItem{}, nil
	}

	var parsed metaRecommendationResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return []RecommendedItem{}, nil
	}
	return parsed.Results, nil
}
