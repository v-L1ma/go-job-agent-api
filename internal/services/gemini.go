package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/genai"
)

type GeminiResponseWrapper struct {
    Respostas []Answer `json:"respostas"`
}

type Answer struct {
    Pergunta string `json:"pergunta"`
    Resposta string `json:"resposta"`
}

var client *genai.Client

func init() {
	var err error
	client, err = genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})

	if err != nil {
		fmt.Printf("failed to initialize gemini client: %v\n", err)
	}
}

// func BuildPrompt(questions []Question) string {
// 	var sb strings.Builder

// 	sb.WriteString("Retorne um json no formato {respostas:[{pergunta:'texto pergunta', resposta:'texto resposta'}]} respondendo as seguintes perguntas com dados ficticios:\n")
// 	sb.WriteString("Não precisa formatar o json de resposta, apenas retorne o json, não simule quebra de linha:\n")
// 	sb.WriteString("Perguntas:\n")
// 	data, err := json.Marshal(questions)
// 	if err != nil {
// 		fmt.Printf("failed to marshal questions: %v\n", err)
// 		return ""
// 	}
// 	sb.Write(data)

// 	return sb.String()
// }

func ParseGeminiResponse(rawResponse string) (GeminiResponseWrapper, error) {
    var wrapper GeminiResponseWrapper
    err := json.Unmarshal([]byte(rawResponse), &wrapper)
    if err != nil {
        return GeminiResponseWrapper{}, fmt.Errorf("erro ao converter JSON: %v", err)
    }

    return wrapper, nil
}

var models = []string{
    "gemma-4-31b-it",
    "gemini-3.5-flash",
    "gemini-3.1-flash-lite",
    "gemini-3-flash-preview",
    "gemini-2.5-flash",
}

func callModel(ctx context.Context, model, prompt string) (string, error) {
    response, err := client.Models.GenerateContent(
        ctx,
        model,
        genai.Text(prompt),
        &genai.GenerateContentConfig{
            ThinkingConfig: &genai.ThinkingConfig{
                ThinkingLevel: genai.ThinkingLevelHigh,
            },
            Tools: []*genai.Tool{
                {CodeExecution: &genai.ToolCodeExecution{}},
            },
        },
    )

    if err != nil {
        return "", err
    }

    if len(response.Candidates) == 0 {
        return "", fmt.Errorf("no candidates")
    }

    if len(response.Candidates[0].Content.Parts) == 0 {
        return "", fmt.Errorf("empty response")
    }

    text := response.Candidates[0].Content.Parts[len(response.Candidates)].Text
    fmt.Printf("[DEBUG] model=%s response_text=%s\n", model, text)
    return text, nil
}

func GenerateResponse(prompt string) (any, error) {
	if client == nil {
		return GeminiResponseWrapper{}, errors.New("gemini client not initialized")
	}

	fmt.Printf("Generated Prompt:\n%s\n", prompt)

	ctx := context.Background()

	var errs []error

    const maxRetries = 3

	for _, model := range models {

		for retry := 1; retry <= maxRetries; retry++ {

			response, err := callModel(ctx, model, prompt)

			if err == nil {
				return response, nil
			}

			log.Printf(
				"model=%s retry=%d error=%v",
				model,
				retry,
				err,
			)

			errs = append(errs, fmt.Errorf("%s: %w", model, err))

			time.Sleep(time.Duration(retry) * time.Second)
		}
	}
    return nil, errors.Join(errs...)
}

func debugPrint[T any](r *T) {

	response, err := json.MarshalIndent(*r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(response))
}

func GenerateEmbeddings(input string, model string) (string, error) {
	if client == nil {
		return "", errors.New("gemini client not initialized")
	}

	if input == "" {
		return "", errors.New("input text is empty")
	}

	ctx := context.Background()

	contents := []*genai.Content{
		genai.NewContentFromText(input, genai.RoleUser),
	}
	result, err := client.Models.EmbedContent(ctx,
		model,
		contents,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("gemini embed content: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return "", errors.New("no embeddings returned")
	}

	values := result.Embeddings[0].Values
	var buf strings.Builder
	buf.WriteByte('[')
	for i, v := range values {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	buf.WriteByte(']')

	return buf.String(), nil
}