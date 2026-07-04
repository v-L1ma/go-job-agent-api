package helpers

import (
	"encoding/json"
	"errors"
	"job-agent-api/internal/dto"
	"strconv"
	"strings"
)

func ParseInt32(s string) (int32, error) {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}

func cleanJSON(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

func ParseQuestionsResponse(raw string) (dto.RespostasRequest, error) {
	cleaned := cleanJSON(raw)

	var flat map[string]string
	if err := json.Unmarshal([]byte(cleaned), &flat); err == nil && len(flat) > 0 {
		return flatToRespostas(flat), nil
	}

	var data dto.RespostasRequest
	if err := json.Unmarshal([]byte(cleaned), &data); err == nil && len(data.Respostas) > 0 {
		return data, nil
	}

	cleaned = strings.ReplaceAll(raw, `\n`, "\n")
	cleaned = strings.ReplaceAll(cleaned, `\"`, `"`)
	cleaned = cleanJSON(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &flat); err == nil && len(flat) > 0 {
		return flatToRespostas(flat), nil
	}

	if err := json.Unmarshal([]byte(cleaned), &data); err == nil && len(data.Respostas) > 0 {
		return data, nil
	}

	return dto.RespostasRequest{}, errors.New("não foi possível interpretar a resposta da IA")
}

func flatToRespostas(flat map[string]string) dto.RespostasRequest {
	var respostas []dto.RespostaQuestao
	for pergunta, resposta := range flat {
		respostas = append(respostas, dto.RespostaQuestao{
			Pergunta: pergunta,
			Resposta: resposta,
		})
	}
	return dto.RespostasRequest{Respostas: respostas}
}
