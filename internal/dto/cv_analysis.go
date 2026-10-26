package dto

import (
	"encoding/json"
	"strings"
)

// CvAnalysis é o objeto "analysis" devolvido pelas rotas de análise de
// currículo (pública e privadas). O contrato é camelCase, com duas
// exceções propositais:
//   - "cv": reutiliza o shape PascalCase de GeneratedCv (mesmo objeto que o
//     frontend já renderiza no viewer de currículo);
//   - "job": null quando nenhuma vaga foi informada.
type CvAnalysis struct {
	Score       int                      `json:"score"`
	Summary     string                   `json:"summary"`
	Categories  []CvAnalysisCategory     `json:"categories"`
	Strengths   []CvAnalysisStrength     `json:"strengths"`
	Improvements []CvAnalysisImprovement `json:"improvements"`
	Job         *CvAnalysisJob           `json:"job"`
	Cv          GeneratedCv              `json:"cv"`
}

// CvAnalysisCategory é um eixo de avaliação (4 a 6 por análise) com score
// parcial de 0 a 100.
type CvAnalysisCategory struct {
	Name    string `json:"name"`
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

// CvAnalysisStrength é um ponto forte ancorado na seção do documento. Area
// usa o MESMO conjunto fechado de CvAnalysisImprovement.Area (estrutura |
// contato | resumo | skills | experiencias | educacao | geral): o viewer do
// currículo usa para posicionar/highlightar a nota na seção exata. Text é a
// frase do ponto forte.
type CvAnalysisStrength struct {
	Area string `json:"area"`
	Text string `json:"text"`
}

// UnmarshalJSON aceita tanto o formato atual ({area, text}) quanto o formato
// legado (string pura), que era o contrato antigo de "strengths". A coerção
// da string legada para {area, text} acontece aqui para que o caminho de
// leitura do banco (GET /cv-analysis/:analysisId, que faz Unmarshal direto
// em CvAnalysis) também normalize análises antigas persistidas como
// ["forte1", "forte2"] — sem isso, o Unmarshal falharia e o GET retornaria
// 500. A heurística tenta inferir a seção pelo conteúdo; sem match, "geral".
// Itens de tipo inesperado (ex.: number) viram zero-value e são descartados
// depois em normalizeCvAnalysis — um item ruim nunca derruba o parse do
// objeto inteiro.
func (s *CvAnalysisStrength) UnmarshalJSON(data []byte) error {
	// Formato legado: apenas uma string.
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		s.Area = inferStrengthArea(text)
		s.Text = text
		return nil
	}

	// Formato atual: objeto {area, text}. Tipo inesperado (ex.: number,
	// array): zera o item em vez de propagar erro.
	var obj struct {
		Area string `json:"area"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		s.Area = ""
		s.Text = ""
		return nil
	}
	s.Area = obj.Area
	s.Text = obj.Text
	return nil
}

// inferStrengthArea aplica uma heurística simples por palavra-chave sobre o
// texto de um ponto forte legado (string pura) para sugerir a seção do
// currículo a que ele pertence. Sem nenhum match, devolve "geral". A ordem
// dos casos importa: o mais específico vem primeiro.
func inferStrengthArea(text string) string {
	value := strings.ToLower(strings.TrimSpace(text))

	switch {
	case strings.Contains(value, "experiênc") || strings.Contains(value, "experienc") ||
		strings.Contains(value, "empresa") || strings.Contains(value, "cargo") ||
		strings.Contains(value, "lider") || strings.Contains(value, "líder"):
		return "experiencias"
	case strings.Contains(value, "formação") || strings.Contains(value, "formacao") ||
		strings.Contains(value, "faculdade") || strings.Contains(value, "universidade") ||
		strings.Contains(value, "curso") || strings.Contains(value, "acadêm") ||
		strings.Contains(value, "academic"):
		return "educacao"
	case strings.Contains(value, "skill") || strings.Contains(value, "ferramenta") ||
		strings.Contains(value, "tecnolog") || strings.Contains(value, "stack") ||
		strings.Contains(value, "linguagem"):
		return "skills"
	case strings.Contains(value, "resumo") || strings.Contains(value, "objetivo") ||
		strings.Contains(value, "perfil"):
		return "resumo"
	case strings.Contains(value, "contato") || strings.Contains(value, "email") ||
		strings.Contains(value, "telefone") || strings.Contains(value, "linkedin"):
		return "contato"
	case strings.Contains(value, "estrutura") || strings.Contains(value, "formatação") ||
		strings.Contains(value, "formatacao") || strings.Contains(value, "layout") ||
		strings.Contains(value, "organiza"):
		return "estrutura"
	default:
		return "geral"
	}
}

// CvAnalysisImprovement é uma melhoria acionável. Area usa um conjunto
// fechado de valores para o frontend mapear a seção do viewer do currículo
// (estrutura | contato | resumo | skills | experiencias | educacao | geral)
// e Severity é alta | media | baixa.
type CvAnalysisImprovement struct {
	Area       string `json:"area"`
	Severity   string `json:"severity"`
	Problem    string `json:"problem"`
	Suggestion string `json:"suggestion"`
}

// CvAnalysisJob é o parecer de fit com uma vaga específica (null quando a
// vaga não foi informada).
type CvAnalysisJob struct {
	Title         string   `json:"title"`
	MatchScore    int      `json:"matchScore"`
	MissingSkills []string `json:"missingSkills"`
	Comment       string   `json:"comment"`
}

// CvAnalysisDetail é a análise persistida: o objeto "analysis" completo com
// os metadados de armazenamento (somente rotas privadas).
type CvAnalysisDetail struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	CvAnalysis
}

// CvAnalysisListItem é a linha do histórico de análises (GET /cv-analysis).
// JobTitle é *string para serializar null quando não havia vaga.
type CvAnalysisListItem struct {
	ID        string  `json:"id"`
	Score     int     `json:"score"`
	Summary   string  `json:"summary"`
	JobTitle  *string `json:"jobTitle"`
	CreatedAt string  `json:"createdAt"`
}
