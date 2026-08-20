# API - Go Job Agent

Base URL: `http://localhost:1323/api/v1`

## Autenticação

Todas as rotas privadas exigem o header:

```
Authorization: Bearer <token>
```

O token JWT é obtido via `POST /login` ou `POST /register` e expira em **15 horas**.

---

## Índice de Rotas

### Públicas (sem autenticação)

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/login` | Autenticar usuário |
| POST | `/register` | Criar nova conta |
| POST | `/refresh-token` | Renovar token JWT |
| POST | `/forgot-password` | Solicitar reset de senha |
| POST | `/reset-password` | Executar reset de senha |

### Privadas (requerem JWT)

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/jobs` | Listar vagas (paginação por cursor) |
| GET | `/jobs/:jobId` | Detalhes de uma vaga |
| POST | `/jobs/:jobId/rate` | Avaliar uma vaga |
| POST | `/jobs/:jobId/cv` | Gerar currículo personalizado para a vaga |
| POST | `/jobs/:jobId/apply` | Candidatar-se a uma vaga |
| POST | `/users/cv` | Upload de currículo (PDF) |
| GET | `/users/cv` | Baixar currículo do usuário (PDF) |
| GET | `/users/cv/generated` | Listar currículos gerados |
| GET | `/users/cv/:cvId` | Baixar currículo gerado por ID (PDF) |
| GET | `/users/profile` | Obter perfil do usuário |
| PUT | `/users/profile` | Atualizar perfil |
| POST | `/users/preferences` | Criar/atualizar preferências |
| GET | `/users/preferences` | Obter preferências |
| PUT | `/users/change-password` | Alterar senha |
| GET | `/users/statistics` | Obter estatísticas do dashboard |

### Admin (requerem JWT + role `Admin`)

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/admin/roles` | Listar todos os cargos |
| GET | `/admin/roles/:id` | Obter um cargo por ID |
| POST | `/admin/roles` | Criar novo cargo |
| PUT | `/admin/roles/:id` | Atualizar um cargo |
| DELETE | `/admin/roles/:id` | Desativar um cargo (soft delete) |
| GET | `/admin/scrapper/logs/:searchQueryId` | Logs do scraper de uma search query |
| GET | `/admin/scrapper/jobs/:executionId` | Vagas de uma execução do scraper |
| GET | `/admin/search-queries` | Listar search queries |

---

## Rotas Públicas

### POST `/login`

Autentica o usuário e retorna um token JWT.

**Request:**
```json
{
  "email": "usuario@email.com",
  "password": "minhaSenha123"
}
```

**Response `200 OK`:**
```json
{
  "message": "Login efetuado com sucesso!",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response `400 Bad Request`:**
```json
{
  "error": "Email e/ou senha inválidos."
}
```

---

### POST `/register`

Cria uma nova conta de usuário.

**Request:**
```json
{
  "name": "João Silva",
  "email": "joao@email.com",
  "password": "senha123",
  "confirmPassword": "senha123"
}
```

Validações:
- `name`: obrigatório, max 60 caracteres
- `email`: obrigatório, formato email, max 50 caracteres
- `password`: obrigatório, min 6, max 50 caracteres
- `confirmPassword`: obrigatório, min 6 caracteres

**Response `201 Created`:**
```json
{
  "Message": "Usuário criado com sucesso."
}
```

**Response `400 Bad Request` (validação):**
```json
{
  "errors": {
    "RegisterRequest.email": "Email deve ser um endereço de e-mail válido"
  }
}
```

**Response `400 Bad Request`:**
```json
{
  "error": "Este e-mail já está em uso."
}
```

---

### POST `/refresh-token`

Renova um token JWT existente antes da expiração.

**Request:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response `200 OK`:**
```json
{
  "message": "Token renovado com sucesso!",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response `401 Unauthorized`:**
```json
{
  "error": "Token inválido."
}
```

---

### POST `/forgot-password`

Solicita um token para redefinição de senha.

**Request:**
```json
{
  "email": "usuario@email.com"
}
```

**Response `200 OK` (email encontrado):**
```json
{
  "message": "Se o e-mail informado estiver cadastrado, você receberá um link para redefinir sua senha.",
  "resetToken": "a1b2c3d4e5f6..."
}
```

**Response `200 OK` (email não encontrado - mesma mensagem por segurança):**
```json
{
  "message": "Se o e-mail informado estiver cadastrado, você receberá um link para redefinir sua senha."
}
```

> O `resetToken` retornado é um hex de 64 caracteres que expira em 1 hora. Em produção, este token deve ser enviado por email.

---

### POST `/reset-password`

Redefine a senha usando o token obtido em `/forgot-password`.

**Request:**
```json
{
  "token": "a1b2c3d4e5f6...",
  "newPassword": "novaSenha123",
  "confirmPassword": "novaSenha123"
}
```

Validações:
- `token`: obrigatório
- `newPassword`: obrigatório, min 6, max 50
- `confirmPassword`: obrigatório, min 6

**Response `200 OK`:**
```json
{
  "message": "Senha redefinida com sucesso!"
}
```

**Response `400 Bad Request`:**
```json
{
  "error": "Token inválido ou expirado."
}
```

---

## Rotas Privadas

Todas as rotas abaixo exigem o header:

```
Authorization: Bearer <token>
```

---

### GET `/jobs`

Lista vagas disponíveis com paginação por cursor.

**Query Parameters:**
| Parâmetro | Tipo | Obrigatório | Padrão | Descrição |
|-----------|------|-------------|--------|-----------|
| `limit` | int | não | `10` | Quantidade de resultados por página |
| `cursor` | string (RFC3339) | não | - | Timestamp do último item da página anterior |

**Response `200 OK`:**
```json
{
  "jobs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "plataformJobId": "12345",
      "title": "Desenvolvedor Go",
      "description": "Descrição da vaga...",
      "url": "https://exemplo.com/vaga/12345",
      "isApplied": false,
      "status": "pending",
      "active": true,
      "createdBy": "system",
      "createdAt": "2026-07-03T10:00:00Z",
      "lastModifiedBy": "system",
      "lastModifiedAt": "2026-07-03T10:00:00Z",
      "platform": "LinkedIn",
      "company": "Empresa XYZ"
    }
  ],
  "nextCursor": "2026-07-03T10:00:00Z"
}
```

Para obter a próxima página, use o valor de `nextCursor` como `?cursor=2026-07-03T10:00:00Z`.

---

### GET `/jobs/:jobId`

Retorna os detalhes de uma vaga específica.

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `jobId` | UUID | ID da vaga |

**Response `200 OK`:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Desenvolvedor Go",
  "description": "Descrição da vaga...",
  "url": "https://exemplo.com/vaga/12345",
  "status": "pending",
  "active": true,
  "platform": "LinkedIn",
  "company": "Empresa XYZ",
  "plataform_job_id": "12345",
  "is_applied": false,
  "created_by": "system",
  "created_at": {...},
  "last_modified_by": "system",
  "last_modified_at": {...}
}
```

> O response retorna o struct `sqlc.Job` diretamente (nomes dos campos em snake_case).

---

### POST `/jobs/:jobId/rate`

Avalia uma vaga (curtir/não curtir).

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `jobId` | UUID | ID da vaga |

**Request:**
```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "liked": true,
  "feedback": "Vaga muito interessante!"
}
```

**Response `200 OK`:**
```json
{
  "message": "Muito obrigado pela sua avaliação!"
}
```

**Response `400 Bad Request` (já avaliada):**
```json
{
  "error": "Você já avaliou esta vaga."
}
```

---

### POST `/jobs/:jobId/cv`

Gera um currículo personalizado com base na vaga e no currículo do usuário.

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `jobId` | UUID | ID da vaga |

**Response `200 OK`:** PDF binário (`application/pdf`)

**Response `404 Not Found`:**
```json
{
  "error": "A vaga informada não foi encontrada"
}
```
ou
```json
{
  "error": "O curriculo do usuario informado não foi encontrado"
}
```

> O usuário deve ter feito upload de um currículo via `POST /users/cv` antes de usar esta rota.

---

### POST `/jobs/:jobId/apply`

Candidata-se a uma vaga.

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `jobId` | UUID | ID da vaga |

**Response `201 Created`:**
```json
{
  "message": "Aplicação concluída com sucesso!"
}
```

---

### POST `/users/cv`

Faz upload de um currículo em PDF. O arquivo é extraído e analisado por IA.

**Request:** Multipart form-data
| Campo | Tipo | Descrição |
|-------|------|-----------|
| `cv` | file | Arquivo PDF do currículo |

**Response `200 OK`:**
```json
{
  "filename": "curriculo.pdf",
  "size": 123456,
  "type": "application/pdf",
  "content": "Texto extraído do PDF...",
  "response": {
    "Nome": "João Silva",
    "Email": "joao@email.com",
    "Telefone": "(11) 99999-9999",
    "Linkedin": "https://linkedin.com/in/joao",
    "Github": "https://github.com/joao",
    "Resumo": "Desenvolvedor Go com 5 anos de experiência...",
    "Skills": ["Go", "PostgreSQL", "Docker"],
    "Experiencias": [
      {
        "Cargo": "Desenvolvedor Backend",
        "Empresa": "Empresa XYZ",
        "DataInicio": "2020-01",
        "DataFim": "2023-12",
        "Descricao": "Desenvolvimento de APIs..."
      }
    ],
    "Educacao": [
      {
        "Curso": "Ciência da Computação",
        "Instituicao": "Universidade XYZ",
        "DataInicio": "2015-01",
        "DataFim": "2019-12"
      }
    ]
  }
}
```

**Response `400 Bad Request`:**
```json
{
  "error": "Currículo não enviado"
}
```

---

### GET `/users/cv`

Baixa o currículo do usuário como PDF.

**Response `200 OK`:** PDF binário (`application/pdf`)

**Response `404 Not Found`:**
```json
{
  "error": "O curriculo do usuario informado não foi encontrado"
}
```

---

### GET `/users/cv/generated`

Lista todos os currículos gerados pelo usuário.

**Response `200 OK`:**
```json
{
  "message": "Curriculos listados com sucesso",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "...",
      "job_id": "...",
      "file_name": "Curriculo-Desenvolvedor Go",
      "extracted_text": "...",
      "active": true,
      "created_by": "...",
      "created_at": "...",
      "last_modified_by": "...",
      "last_modified_at": "..."
    }
  ]
}
```

---

### GET `/users/cv/:cvId`

Baixa um currículo gerado específico como PDF.

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `cvId` | UUID | ID do currículo gerado |

**Response `200 OK`:** PDF binário (`application/pdf`)

**Response `404 Not Found`:**
```json
{
  "error": "O curriculo informado não foi encontrado"
}
```

**Response `400 Bad Request` (outro usuário):**
```json
{
  "error": "Este curriculo pertence a outro usuário"
}
```

---

### GET `/users/profile`

Retorna os dados do perfil do usuário autenticado.

**Response `200 OK`:**
```json
{
  "message": "Perfil encontrado com sucesso!",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "João Silva",
    "email": "joao@email.com",
    "cpf": ""
  }
}
```

---

### PUT `/users/profile`

Atualiza nome e/ou email do perfil.

**Request:**
```json
{
  "name": "João Silva Atualizado",
  "email": "joaonovo@email.com"
}
```

Validações:
- `name`: min 6, max 50 caracteres
- `email`: formato email, min 6, max 50 caracteres

**Response `200 OK`:**
```json
{
  "message": "Perfil atualizado com sucesso!"
}
```

**Response `400 Bad Request`:**
```json
{
  "error": "Este e-mail já está em uso."
}
```

---

### POST `/users/preferences`

Cria ou atualiza as preferências de busca de vagas do usuário (skills e níveis de senioridade).

**Request:**
```json
{
  "skills": ["Go", "PostgreSQL", "Docker", "Kubernetes"],
  "levels": ["Junior", "Pleno", "Senior"]
}
```

> Se as preferências já existirem, serão atualizadas. Caso contrário, serão criadas.

**Response `200 OK` (criação):**
```json
{
  "message": "Preferências criadas com sucesso!"
}
```

**Response `200 OK` (atualização):**
```json
{
  "message": "Preferências atualizadas com sucesso!"
}
```

**Response `400 Bad Request`:**
```json
{
  "error": "Informe ao menos uma Habilidade e uma senioridade."
}
```

---

### GET `/users/preferences`

Retorna as preferências de busca do usuário.

**Response `200 OK`:**
```json
{
  "message": "Preferências encontradas com sucesso!",
  "data": [
    {
      "UserId": "550e8400-e29b-41d4-a716-446655440000",
      "Skills": ["Go", "PostgreSQL", "Docker"],
      "Levels": ["Pleno", "Senior"]
    }
  ]
}
```

---

### PUT `/users/change-password`

Altera a senha do usuário autenticado.

**Request:**
```json
{
  "currentPassword": "senhaAntiga123",
  "newPassword": "senhaNova456",
  "confirmNewPassword": "senhaNova456"
}
```

**Response `200 OK`:**
```json
{
  "message": "Senha alterada com sucesso!"
}
```

**Response `400 Bad Request`:**
```json
{
  "error": "Senha atual incorreta."
}
```

---

### GET `/users/statistics`

Retorna estatísticas do dashboard do usuário.

**Response `200 OK`:**
```json
{
  "message": "Estatísticas encontradas com sucesso!",
  "data": {
    "total": {
      "count": 150,
      "variation": 20,
      "variationLabel": "vs mês passado"
    },
    "applied": {
      "count": 45,
      "successRate": 30
    },
    "skipped": {
      "count": 105,
      "label": "Filtros aplicados"
    },
    "failures": {
      "count": 0,
      "thisWeek": 0
    },
    "applicationsPerDay": [
      { "date": "01/07", "count": 3 },
      { "date": "02/07", "count": 5 }
    ],
    "platformDistribution": [
      { "platform": "LinkedIn", "count": 80 },
      { "platform": "Indeed", "count": 70 }
    ],
    "statusDistribution": [
      { "status": "Total", "count": 150 },
      { "status": "Aplicadas", "count": 45, "percentage": 30 },
      { "status": "Puladas", "count": 105, "percentage": 70 },
      { "status": "Falhas", "count": 0, "percentage": 0 }
    ],
    "recentApplications": []
  }
}
```

---

## Rotas Admin

Todas as rotas abaixo exigem o header:

```
Authorization: Bearer <token>
```

E o usuário autenticado deve possuir a role `Admin`. Caso contrário, retorna `403 Forbidden`.

---

### GET `/admin/roles`

Lista todos os cargos cadastrados (ativos e inativos).

**Response `200 OK`:**
```json
{
  "message": "Cargos listados com sucesso!",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Admin",
      "active": true,
      "createdBy": "550e8400-e29b-41d4-a716-446655440000",
      "createdAt": "2026-07-03T10:00:00Z",
      "lastModifiedBy": "550e8400-e29b-41d4-a716-446655440000",
      "lastModifiedAt": "2026-07-03T10:00:00Z"
    }
  ]
}
```

---

### GET `/admin/roles/:id`

Retorna um cargo específico.

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `id` | UUID | ID do cargo |

**Response `200 OK`:**
```json
{
  "message": "Cargo encontrado com sucesso!",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Standard",
    "active": true,
    "createdBy": "550e8400-e29b-41d4-a716-446655440000",
    "createdAt": "2026-07-03T10:00:00Z",
    "lastModifiedBy": "550e8400-e29b-41d4-a716-446655440000",
    "lastModifiedAt": "2026-07-03T10:00:00Z"
  }
}
```

**Response `404 Not Found`:**
```json
{
  "error": "Cargo não encontrado"
}
```

---

### POST `/admin/roles`

Cria um novo cargo.

**Request:**
```json
{
  "name": "Recruiter"
}
```

Validações:
- `name`: obrigatório, máx. 50 caracteres

**Response `201 Created`:**
```json
{
  "message": "Cargo criado com sucesso!"
}
```

**Response `409 Conflict` (nome já existente):**
```json
{
  "error": "Cargo já existe"
}
```

**Response `400 Bad Request` (validação):**
```json
{
  "errors": {
    "name": "Este campo é obrigatório"
  }
}
```

---

### PUT `/admin/roles/:id`

Atualiza o nome e status de um cargo.

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `id` | UUID | ID do cargo |

**Request:**
```json
{
  "name": "Senior Recruiter",
  "active": true
}
```

> Atenção: `active` é obrigatório no body (bool). Enviar `false` desativa o cargo.

**Response `200 OK`:**
```json
{
  "message": "Cargo atualizado com sucesso!"
}
```

**Response `404 Not Found`:**
```json
{
  "error": "Cargo não encontrado"
}
```

**Response `409 Conflict` (nome em uso por outro cargo):**
```json
{
  "error": "Já existe um cargo com este nome"
}
```

---

### DELETE `/admin/roles/:id`

Desativa um cargo (soft delete). O registro permanece no banco com `active = false`.

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `id` | UUID | ID do cargo |

**Response `200 OK`:**
```json
{
  "message": "Cargo desativado com sucesso!"
}
```

**Response `404 Not Found`:**
```json
{
  "error": "Cargo não encontrado"
}
```

---

## Middleware Global

- **RequestLogger** — Loga todas as requisições
- **Recover** — Recupera de panics
- **CORS** — Configurável via `CORS_ORIGINS` (padrão: `http://localhost:3000`)

## Códigos de Erro Comuns

| Código | Significado |
|--------|-------------|
| `400` | Bad Request — corpo inválido, validação, duplicidade |
| `401` | Unauthorized — token ausente, inválido ou expirado |
| `403` | Forbidden — usuário sem a role `Admin` |
| `404` | Not Found — recurso não encontrado |
| `409` | Conflict — recurso duplicado (ex.: cargo com nome existente) |
| `500` | Internal Server Error — erro interno do servidor |

## Variáveis de Ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_USER` | `jacob` | Usuário do banco |
| `DB_PASSWORD` | `password` | Senha do banco |
| `DB_NAME` | `bookstoreDB` | Nome do banco |
| `JWT_SECRET` | - | Chave secreta para assinar tokens JWT |
| `CORS_ORIGINS` | `http://localhost:3000` | Origens permitidas (separadas por vírgula) |
| `GEMINI_API_KEY` | - | Chave da API Google Gemini |
