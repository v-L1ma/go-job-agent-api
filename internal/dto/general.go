package dto

type ResponseBase[T any] struct {
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
}

type GeneratedCv struct {
	Nome         string
	Email        string
	Telefone     string
	Linkedin     string
	Github       string
	Resumo       string
	Skills       []string
	Experiencias []Experiencia
	Educacao     []Educacao
	// SecoesExtras guarda tudo que não cabe nos campos fixos acima (Idiomas,
	// Certificações, Projetos, Cursos, Prêmios, Objetivos, Informações
	// adicionais etc.): uma entrada por seção, com o título exatamente como
	// no documento e o conteúdo integral. JSONs antigos não têm a chave →
	// nil, sem quebrar a leitura (compatível retro).
	SecoesExtras []SecaoExtra
}

// SecaoExtra é uma seção adicional do currículo fora do schema fixo, para
// garantir que nenhuma parte do documento se perca na estruturação.
type SecaoExtra struct {
	Titulo   string
	Conteudo string
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
