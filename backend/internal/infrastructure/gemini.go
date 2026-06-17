package infrastructure

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

type GeminiClient struct {
	apiKey string
}

func NewGeminiClient() (*GeminiClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}
	return &GeminiClient{apiKey: apiKey}, nil
}

func (g *GeminiClient) GenerateProductDescription(ctx context.Context, title, keywords string) (string, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create gemini client: %w", err)
	}

	prompt := fmt.Sprintf(
		"あなたはフリマアプリの商品説明文を書くプロです。以下の情報をもとに、購買意欲を高める魅力的な商品説明文を150〜250文字で生成してください。\n\n商品名: %s\nキーワード: %s\n\n説明文のみを出力してください。",
		title, keywords,
	)

	result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	text := result.Text()
	if text == "" {
		return "", fmt.Errorf("no content generated")
	}

	return text, nil
}

func (g *GeminiClient) AnswerProductQuestion(ctx context.Context, title, description, question string) (string, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create gemini client: %w", err)
	}

	prompt := fmt.Sprintf(
		"あなたはフリマアプリの商品に関する質問に答えるアシスタントです。以下の商品情報をもとに、購入希望者からの質問に簡潔かつ親切に答えてください。商品情報からは判断できない質問の場合は、その旨を伝えて出品者へ直接問い合わせるよう案内してください。\n\n商品名: %s\n商品説明: %s\n\n質問: %s\n\n回答のみを出力してください。",
		title, description, question,
	)

	result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	text := result.Text()
	if text == "" {
		return "", fmt.Errorf("no content generated")
	}

	return text, nil
}
