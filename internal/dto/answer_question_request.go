package dto

type QuestaoPersonalizada struct {
    Pergunta string   `json:"pergunta"`
    Tipo     string   `json:"tipo"`
    Opcoes   []string `json:"opcoes,omitempty"`
}

type QuestoesRequest struct {
    JobId    string   `json:"jobId"`
    Questoes []QuestaoPersonalizada `json:"questoes"`
}

type RespostaQuestao struct {
    Pergunta string `json:"pergunta"`
    Resposta string `json:"resposta"`
}

type RespostasRequest struct {
    Respostas []RespostaQuestao `json:"respostas"`
}