package handlers

import (
	"encoding/json"
	"job-agent-api/internal/database"
	"job-agent-api/internal/dto"
	"job-agent-api/internal/helpers"
	sqlc "job-agent-api/internal/queries"
	"job-agent-api/internal/services"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
)

func GetQuestions(c *echo.Context, db *database.Database) error {
	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	err := userID.Scan(claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	existUser, err := db.Query.ExistsUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !existUser {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Usuário não encontrado."})
	}

	questions, err := db.Query.GetUserQuestions(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, dto.ResponseBase[[]sqlc.GetUserQuestionsRow]{
		Message: "Perguntas encontradas com sucesso!",
		Data:    questions,
	})
}

func EditQuestion(c *echo.Context, db *database.Database) error {
	id := c.Param("questionId")
	var questionID pgtype.UUID
	if err := questionID.Scan(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	err := userID.Scan(claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	existUser, err := db.Query.ExistsUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if !existUser {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Usuário não encontrado."})
	}

	question, err := db.Query.GetQuestionById(c.Request().Context(), questionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if question.UserId != userID {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Questão não pertence a outro usuário."})
	}

	var req dto.EditQuestion
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err = db.Query.UpdateQuestionAnswer(c.Request().Context(), sqlc.UpdateQuestionAnswerParams{
		Id: questionID,
		Answer: req.Answer,
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"Message": "Perguntas encontradas com sucesso!",})
}

func AnswerQuestion(c *echo.Context, db *database.Database) error {

	claims := c.Get("user").(*services.Claims)
	var userID pgtype.UUID
	if err := userID.Scan(claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req dto.QuestoesRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var jobID pgtype.UUID
	if err := jobID.Scan(req.JobId); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	job, err := db.Query.GetJobById(c.Request().Context(), jobID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	questions := make([]string, len(req.Questoes))

	for i, question := range req.Questoes {
		questions[i] = question.Pergunta
	}
		
	dbAnswer, err := db.Query.FindQuestionAnswer(c.Request().Context(), questions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao buscar respostas no banco",
		})
	}

	answerMap := make(map[string]bool)

	for _, answer := range dbAnswer {
		answerMap[answer.Question] = true
	}

	filtered := make([]dto.QuestaoPersonalizada, 0, len(req.Questoes))
	for _, q := range req.Questoes {
		if !answerMap[q.Pergunta] {
			filtered = append(filtered, q)
		}
	}

	if len(filtered) == 0 {
		respostasDTO := make([]dto.RespostaQuestao, len(dbAnswer))
		for i, a := range dbAnswer {
			respostasDTO[i] = dto.RespostaQuestao{
				Pergunta: a.Question,
				Resposta: a.Answer,
			}
		}
		return c.JSON(http.StatusOK, map[string]any{
			"message": "Respostas recebidas com sucesso!",
			"data":    respostasDTO,
		})
	}

	userCv, err := db.Query.GetUserCv(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	questoesJSON, err := json.Marshal(filtered)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var sb strings.Builder

	sb.WriteString("Você é um especialista em recrutamento, RH e otimização de currículos.")
	sb.WriteString("Sua função é responder formulários de candidatura de forma estratégica, valorizando ao máximo o perfil do candidato e aumentando suas chances de avançar no processo seletivo.")
	sb.WriteString("Utilize o currículo, a descrição da vaga e o contexto das perguntas para produzir as melhores respostas possíveis.")
	sb.WriteString("")

	sb.WriteString("Regras obrigatórias:")
	sb.WriteString("- Retorne APENAS um JSON válido.")
	sb.WriteString("- Nunca escreva qualquer texto fora do JSON.")
	sb.WriteString("- Nunca utilize markdown.")
	sb.WriteString("")
	sb.WriteString("- Cada resposta deve parecer escrita pelo próprio candidato, em linguagem natural, profissional e confiante.")
	sb.WriteString("- Valorize as experiências, competências e resultados apresentados no currículo.")
	sb.WriteString("- Sempre adapte as respostas ao contexto da vaga.")
	sb.WriteString("- Quando houver mais de uma interpretação possível, escolha a que mais favoreça o candidato, desde que seja coerente com o currículo.")
	sb.WriteString("- Se uma pergunta possuir opções de resposta, escolha a opção que melhor representa o perfil do candidato.")
	sb.WriteString("- Se a pergunta for aberta, produza uma resposta completa, convincente e bem escrita.")
	sb.WriteString("- Evite respostas curtas quando uma resposta mais detalhada agregar valor.")
	sb.WriteString("")
	sb.WriteString("Informações ausentes:")
	sb.WriteString("- Nunca responda 'Informação não disponível no currículo', 'Não informado', 'Não sei' ou frases semelhantes.")
	sb.WriteString("- Quando o currículo não fornecer a informação exata, faça uma inferência razoável utilizando todo o contexto disponível.")
	sb.WriteString("- Considere a experiência profissional, projetos, tecnologias utilizadas, formação, habilidades e principalmente a descrição da vaga.")
	sb.WriteString("- Caso uma informação específica realmente não exista, construa uma resposta plausível que demonstre interesse, capacidade de aprendizado e alinhamento com a vaga.")
	sb.WriteString("- Não invente fatos objetivos como empresas onde trabalhou, certificados, graduações, idiomas, salários, datas, tempo de experiência ou tecnologias que nunca aparecem no currículo.")
	sb.WriteString("- É permitido inferir competências comportamentais (proatividade, colaboração, aprendizado rápido, organização, comunicação, resolução de problemas, adaptabilidade) quando elas forem compatíveis com a trajetória apresentada.")
	sb.WriteString("")
	sb.WriteString("Perguntas de declaração pessoal, compliance e conflitos de interesse:")
	sb.WriteString("- Algumas perguntas tratam de informações pessoais, conflitos de interesse, vínculos profissionais, acessibilidade, fornecedores, parentesco ou declarações legais.")
	sb.WriteString("- Essas perguntas NÃO devem ser respondidas utilizando criatividade ou tentando valorizar o candidato.")
	sb.WriteString("- Utilize apenas informações explicitamente presentes no currículo ou fornecidas pelo usuário.")
	sb.WriteString("- Quando a resposta não puder ser determinada a partir das informações fornecidas, utilize uma resposta neutra e conservadora.")
	sb.WriteString("- Nunca invente vínculos, parentescos, empresas, negócios próprios, sociedades, conflitos de interesse ou qualquer informação pessoal.")
	sb.WriteString("- Para perguntas de Sim/Não relacionadas a compliance, conflito de interesses ou declarações pessoais, utilize 'Não' apenas quando não houver qualquer indício de que a resposta seja 'Sim'.")
	sb.WriteString("- Para perguntas abertas desse tipo, quando não houver informações suficientes, responda de forma apropriada ao contexto, por exemplo: 'Não possuo informações adicionais a declarar no momento.' ou deixe o campo vazio caso isso seja mais adequado ao formulário.")
	sb.WriteString("- Nunca gere respostas que possam representar uma declaração falsa sobre a vida pessoal do candidato.")
	sb.WriteString("Objetivo:")
	sb.WriteString("- As respostas devem aumentar as chances do candidato ser selecionado para a próxima etapa.")
	sb.WriteString("- Priorize respostas positivas, profissionais, coerentes e persuasivas.")
	sb.WriteString("- Nunca contradiga informações do currículo.")
	sb.WriteString("- Seja o mais completo possível.")

	sb.WriteString("Currículo:")
	sb.WriteString("```")
	sb.WriteString(userCv.ExtractedText)
	sb.WriteString("```")

	sb.WriteString("Descrição da vaga:")
	sb.WriteString("```")
	sb.WriteString(job.Description)
	sb.WriteString("```")

	sb.WriteString("Perguntas:")
	sb.WriteString("```")
	sb.WriteString(string(questoesJSON))
	sb.WriteString("```")

	sb.WriteString("Sua prioridade é responder como se fosse o candidato, adaptando o currículo às necessidades da vaga.")
	sb.WriteString("Níveis de evidência:")

	sb.WriteString("1. Explícito no currículo: Responda normalmente.")

	sb.WriteString("2. Pode ser inferido do currículo: Responda com confiança, sem mencionar que foi uma inferência.")

	sb.WriteString("3. Não está no currículo, mas é coerente com a vaga: Produza uma resposta profissional baseada em motivação, capacidade de aprendizado e boas práticas de mercado.")

	sb.WriteString("4. Nunca invente:")
	sb.WriteString("- empresas")
	sb.WriteString("- cargos")
	sb.WriteString("- certificados")
	sb.WriteString("- formações")
	sb.WriteString("- datas")
	sb.WriteString("- salários")
	sb.WriteString("- anos de experiência")
	sb.WriteString("- idiomas")
	sb.WriteString("- tecnologias nunca mencionadas.")
	
	rawResponse, err := services.GenerateResponse(sb.String())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error do gemini": err.Error()})
	}

	rawStr, ok := rawResponse.(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Resposta inválida da IA",
		})
	}

	respostas, err := helpers.ParseQuestionsResponse(rawStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Erro ao analisar resposta da IA: " + err.Error(),
		})
	}

	tx, err := db.Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	qtx := db.Query.WithTx(tx)

	for _, question := range respostas.Respostas {
		err := qtx.CreateQuestion(c.Request().Context(), sqlc.CreateQuestionParams{
			UserId: userID,
			JobId: jobID,
			Question: question.Pergunta,
			Answer: question.Resposta,
			Active: true,
			CreatedBy: userID.String(),
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			LastModifiedBy: userID.String(),
			LastModifiedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return err
		}
	}

	tx.Commit(c.Request().Context())

	mergedMap := make(map[string]string)
	for _, a := range dbAnswer {
		mergedMap[a.Question] = a.Answer
	}
	for _, r := range respostas.Respostas {
		mergedMap[r.Pergunta] = r.Resposta
	}

	merged := make([]dto.RespostaQuestao, 0, len(req.Questoes))
	for _, q := range req.Questoes {
		if ans, ok := mergedMap[q.Pergunta]; ok {
			merged = append(merged, dto.RespostaQuestao{
				Pergunta: q.Pergunta,
				Resposta: ans,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Respostas recebidas com sucesso!",
		"data":    merged,
	})

}