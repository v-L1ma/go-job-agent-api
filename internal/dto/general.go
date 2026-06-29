package dto

type ResponseBase[T any] struct {
	Message string `json:"message,omitempty"`
	Data T `json:"data,omitempty"`
}

type GeneratedCv struct {
	Nome          string
	Email         string
	Telefone      string
	Linkedin      string
	Github        string
	Resumo        string
	Skills        []string
	Experiencias  []Experiencia
	Educacao      []Educacao
}

type Experiencia struct {
	Cargo      string
	Empresa    string
	DataInicio string
	DataFim    string
	Descricao  string
}

type Educacao struct {
	Curso       string
	Instituicao string
	DataInicio  string
	DataFim     string
}