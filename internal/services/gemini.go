package services


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

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
		APIKey:  "",
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

func GenerateResponse(prompt string) (GeminiResponseWrapper, error) {
	if client == nil {
		return GeminiResponseWrapper{}, errors.New("gemini client not initialized")
	}

	fmt.Printf("Generated Prompt:\n%s\n", prompt)

	ctx := context.Background()

	response, err := client.Models.GenerateContent(ctx, 
		"gemma-4-31b-it", 
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

	debugPrint(&response)

	if err != nil {
		return GeminiResponseWrapper{}, fmt.Errorf("failed to generate response: %v", err)
	}

	geminiResponse := response.Candidates[0].Content.Parts[len(response.Candidates)].Text

	respostas, err := ParseGeminiResponse(geminiResponse)
	if err != nil {
		log.Printf("Erro ao processar respostas: %v", err)
	}

	return respostas, nil
}

func debugPrint[T any](r *T) {

	response, err := json.MarshalIndent(*r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(response))
}