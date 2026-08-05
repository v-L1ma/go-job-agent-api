package dto

type ApplicationWithQuestions struct {
	Id             string            `json:"id"`
	Title          string            `json:"title"`
	Company        string            `json:"company"`
	Platform       string            `json:"platform"`
	Status         string            `json:"status"`
	CreatedAt      string            `json:"createdAt"`
	QuestionsCount int32             `json:"questionsCount"`
	Questions      []QuestionAnswer  `json:"questions"`
}

type QuestionAnswer struct {
	Id       string   `json:"id"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Type     string   `json:"type"`
	Options  []string `json:"options"`
}
