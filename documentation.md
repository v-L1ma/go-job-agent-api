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
| GET | `/plans` | Listar planos de assinatura |
| POST | `/webhook/stripe` | Webhook do Stripe (validado por `Stripe-Signature`) |

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
| POST | `/checkout/session/:planId` | Criar sessão de checkout Stripe (client secret) |
| GET | `/subscriptions/me` | Obter assinatura do usuário autenticado |

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
| GET | `/admin/plans` | Listar todos os planos de assinatura (ativos e inativos) |
| GET | `/admin/subscriptions` | Listar todas as assinaturas |
| GET | `/admin/subscriptions/:subscriptionId` | Detalhe de uma assinatura (assinatura + plano + usuário + faturas) |
| GET | `/admin/users/:userId/subscriptions` | Histórico de assinaturas de um usuário |

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

## Pagamentos e Assinaturas (Stripe)

Fluxo completo de assinaturas com Stripe. **Os webhooks são a fonte da
verdade**: status, período e plano da assinatura — e o acesso do usuário
(papéis) — são definidos exclusivamente pelos eventos processados em
`POST /webhook/stripe`.

### GET `/plans` *(pública)*

Lista os planos cadastrados (sincronizados pelos eventos `product.*` /
`price.*` do Stripe).

**Response `200 OK`:**
```json
{
  "message": "Plans listed successfully",
  "data": [
    {
      "Id": "3f6c1b1e-0a1b-4c2d-8e3f-000000000001",
      "Name": "Plano Pro",
      "Description": "Acesso completo",
      "Price": 49.9
    }
  ]
}
```

> `Price` é retornado **em reais (float)** — ex.: `49.9` = R$ 49,90. O banco
> armazena o valor em centavos (inteiro, vindo de `stripe.Price.UnitAmount`,
> ex.: `4990`); a conversão centavos → reais acontece somente na leitura (GET).
> Os caminhos de escrita (webhooks `price.*`/`product.*` e as queries do banco)
> continuam armazenando/gravando em centavos.
> `Description` é `""` quando o plano não tem descrição (`NULL` no banco).

---

### POST `/checkout/session/:planId` *(privada)*

Cria uma Checkout Session (`mode=subscription`) para o plano indicado.

**Path Parameter:**
| Parâmetro | Tipo | Obrigatório | Descrição |
|-----------|------|-------------|-----------|
| `planId` | UUID | sim | ID do plano (tabela `Plan`) |

**Comportamento:**
- Recupera ou cria o Stripe Customer do usuário (tabela `StripeCustomers`) e reutiliza sempre o mesmo;
- Envia `user_id`/`plan_id` na `metadata` da Session **e** da Subscription, além de `client_reference_id` — o webhook identifica o usuário por esses campos (**nunca por e-mail**);
- A ativação do acesso **não** acontece aqui: ela ocorre via webhook (`customer.subscription.created/updated`).

**Response `200 OK`** (`data` = client secret da sessão, ex.: `cs_live_a1b2...`):
```json
{
  "message": "Sessão de pagamento criada com sucesso",
  "data": "cs_live_a1b2c3d4"
}
```

**Response `400 Bad Request`** (parâmetro ausente/inválido ou plano inexistente):
```json
{
  "error": "planId parameter is required"
}
```

**Response `422 Unprocessable Entity`** (plano sem preço Stripe configurado):
```json
{
  "error": "plan does not have a stripe price configured"
}
```

**Response `500 Internal Server Error`**:
```json
{
  "error": "failed to create checkout session"
}
```

---

### GET `/subscriptions/me` *(privada)*

Retorna a assinatura mais recente do usuário autenticado (a query ordena por
`CreatedAt DESC, UpdatedAt DESC LIMIT 1`).

**Response `200 OK` (com assinatura):**
```json
{
  "message": "Assinatura encontrada.",
  "data": {
    "id": "8f0a1b2c-0000-4000-8000-000000000000",
    "planId": "3f6c1b1e-0a1b-4c2d-8e3f-000000000001",
    "status": "active",
    "cancelAtPeriodEnd": false,
    "currentPeriodEnd": "2026-11-08T12:00:00Z",
    "plan": {
      "id": "3f6c1b1e-0a1b-4c2d-8e3f-000000000001",
      "name": "Plano Pro",
      "description": "Acesso completo",
      "price": 49.9
    }
  }
}
```

> `currentPeriodEnd` só é preenchido quando conhecido (`null` caso contrário);
> `plan` é `null` se o plano vinculado não existir mais.
>
> `plan.price` é retornado **em reais (float)** — ex.: `49.9` = R$ 49,90. O
> banco armazena centavos (inteiro, ex.: `4990`); a conversão centavos → reais
> acontece somente na leitura (GET).

**Response `200 OK` (sem assinatura)** — o campo `data` é omitido:
```json
{
  "message": "Nenhuma assinatura encontrada."
}
```

---

### POST `/webhook/stripe` *(pública — chamada pelo Stripe)*

Endpoint de recebimento dos eventos do Stripe. Configure no dashboard do
Stripe (ou via `stripe listen`) os eventos listados abaixo e um
`STRIPE_WEBHOOK_SECRET` **compatível com o endpoint** (`whsec_...`).

**Headers:**
| Header | Obrigatório | Descrição |
|--------|-------------|-----------|
| `Stripe-Signature` | sim | Assinatura calculada pelo Stripe (validada com tolerância de 300s) |

**Fluxo de processamento (por evento):**

1. Validação da assinatura do payload → falha: `400` (nada é processado).
2. `BEGIN` + claim atômico do `event.id` em `StripeWebhookEvents`
   (`INSERT ... ON CONFLICT DO NOTHING RETURNING "Id"`):
   - `pgx.ErrNoRows` (já existia) → `COMMIT` + `200` — duplicata ignorada sem nenhum efeito colateral;
   - evento novo → segue para o handler.
3. Handler do evento executa **na mesma transação** (UPSERTs + concessão/revogação de papel).
4. `COMMIT` → `200`. Qualquer erro → `ROLLBACK` (o claim de idempotência também é desfeito) → `500`, e o **Stripe reenvia** o evento.

**Eventos processados:**
| Evento | Efeito |
|--------|--------|
| `checkout.session.completed` | Relaciona checkout/usuário/plano e grava `checkout_session_id`; cria linha parcial `incomplete` se ainda não existir; **nunca** concede acesso sozinho |
| `customer.subscription.created` | UPSERT da assinatura + sincroniza o papel do plano |
| `customer.subscription.updated` | UPSERT + sincroniza o papel (upgrade/downgrade, `cancel_at_period_end`, `past_due` etc.) |
| `customer.subscription.deleted` | Força `status=canceled` e revoga **apenas** o papel do plano daquela assinatura |
| `invoice.paid` | Registra a cobrança em `StripeInvoices` (`status=paid` + `PaidAt`); não altera a assinatura |
| `invoice.payment_failed` | Registra `status=payment_failed`; **não** cancela a assinatura nem mexe no acesso |
| `price.created`, `price.updated` | Cria/atualiza o plano correspondente |
| `product.created`, `product.updated` | Cria/atualiza o plano (desativa quando o produto está inativo) |
| `product.deleted` | Desativa o plano |

**Eventos que NÃO controlam a assinatura** (claim + `200`, sem efeitos):
`payment_intent.succeeded`, `payment_intent.payment_failed`,
`payment_method.attached` e qualquer outro evento não listado acima.

**Identificação do usuário (nunca por e-mail):**
`metadata[user_id]` → `client_reference_id` → tabela `StripeCustomers`
(mapeamento customer id ↔ usuário).

**Política de acesso (`HasActiveAccess`):**
| Status da assinatura | Acesso |
|----------------------|--------|
| `active`, `trialing`, `past_due` | ✓ concede/mantém o papel do plano |
| `canceled`, `unpaid`, `incomplete`, `incomplete_expired`, `paused`, desconhecido | ✗ revoga o papel do plano |

- `cancelAtPeriodEnd=true` **não** remove o acesso: o status permanece `active` até o fim do período; a revogação definitiva ocorre em `customer.subscription.deleted`.
- A revogação remove **apenas** o papel vinculado ao `Plan.RoleId` da assinatura — papéis `Admin` e papéis de outros planos são preservados.
- A concessão é idempotente (`ON CONFLICT DO NOTHING` em `UserRoles`).

**Respostas:**
| Código | Quando |
|--------|--------|
| `200` | Evento processado, duplicata ignorada ou evento não utilizado |
| `400` | `Stripe-Signature` ausente/inválida ou payload adulterado |
| `500` | Erro interno — transação revertida (claim desfeito) e Stripe reenvia o evento |

**Exemplo de resposta `200`:**
```json
{
  "message": "Assinatura criada/sincronizada."
}
```

---

### Tabelas envolvidas (PostgreSQL)

| Tabela | Papel |
|--------|-------|
| `Subscriptions` | Estado da assinatura (`StripeSubscription_id` UNIQUE, `StripeCheckoutSession_id` UNIQUE, índices por `UserId`) |
| `StripeCustomers` | Mapeamento `UserId` ↔ `StripeCustomerId` (identificação sem e-mail) |
| `StripeInvoices` | Histórico de cobranças (`paid` / `payment_failed`) |
| `StripeWebhookEvents` | Idempotência dos webhooks (`Id` PRIMARY KEY/UNIQUE = `event.id`) |

---

### Rotas admin de assinaturas

Ambas exigem **JWT válido com role `Admin`** (retornam `401` sem token e `403` sem a role).

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/admin/plans` | Lista **TODOS** os planos (ativos e inativos) com campos completos em `camelCase` |
| GET | `/admin/subscriptions` | Lista **todas** as assinaturas com e-mail do usuário (`userEmail`), nome do plano (`plan`) e nome do cargo (`roleName`) |
| GET | `/admin/subscriptions/:subscriptionId` | Detalhe de **uma** assinatura: assinatura + plano + usuário + faturas |
| GET | `/admin/users/:userId/subscriptions` | Histórico de assinaturas de **um** usuário (mais recente primeiro) |

> Diferente do endpoint público `GET /plans` (que retorna apenas planos ativos com campos em PascalCase), `/admin/plans` retorna todos os campos (`stripePriceId`, `roleId`, `active`, `createdAt`, `updatedAt`, etc.) em `camelCase`. Campos nulos no banco (`stripePriceId`, `description`) são emitidos como `""`.
>
> Em ambos os endpoints, quando não há registros o campo `data` é retornado como lista vazia (`[]`), nunca `null` ou ausente.

#### GET `/admin/plans`

**Response `200 OK`:**
```json
{
  "message": "Plans listed successfully",
  "data": [
    {
      "id": "3f6c1b1e-0a1b-4c2d-8e3f-000000000001",
      "name": "Plano Pro",
      "stripePriceId": "price_abc123",
      "description": "Acesso completo",
      "price": 49.9,
      "roleId": "a2238759-645f-4d7e-867e-1ccf3d86afef",
      "active": true,
      "createdAt": "2026-01-02T15:04:05Z",
      "updatedAt": "2026-01-02T15:04:05Z"
    }
  ]
}
```

> `price` é retornado **em reais (float)** — ex.: `49.9` = R$ 49,90. O banco
> armazena centavos (inteiro, ex.: `4990`); a conversão centavos → reais
> acontece somente na leitura (GET).

#### GET `/admin/subscriptions`

`userEmail` vem do usuário relacionado (`LEFT JOIN` em `AspNetUsers`), `plan` e `roleName` vêm do plano/cargo relacionados (`LEFT JOIN` em `Plans`/`Roles`); são `""` quando o registro não existe. `currentPeriodStart`, `currentPeriodEnd` e `canceledAt` são `null` quando ausentes.

**Response `200 OK`:**
```json
{
  "message": "Subscriptions listed successfully",
  "data": [
    {
      "id": "9b2c3d4e-5f6a-4b7c-8d9e-000000000001",
      "userEmail": "joao@mail.com",
      "roleName": "Pro",
      "plan": "Plano Pro",
      "stripeCustomerId": "cus_abc123",
      "stripeSubscriptionId": "sub_xyz789",
      "stripeCheckoutSessionId": "cs_test_abc123",
      "stripePriceId": "price_abc123",
      "status": "active",
      "currentPeriodStart": "2026-01-02T15:04:05Z",
      "currentPeriodEnd": "2026-02-02T15:04:05Z",
      "cancelAtPeriodEnd": false,
      "canceledAt": null,
      "createdAt": "2026-01-02T15:04:05Z",
      "updatedAt": "2026-01-02T15:04:05Z"
    }
  ]
}
```

#### GET `/admin/subscriptions/:subscriptionId`

Detalhe de uma assinatura para o painel admin. **Somente leitura** (os webhooks do Stripe continuam sendo a fonte da verdade de status/período).

O objeto `subscription` tem **exatamente o mesmo shape e os mesmos valores** do item correspondente em `GET /admin/subscriptions` (os 15 campos em `camelCase`).

- `plan` — mesmo shape do `GET /admin/plans` (`id`, `name`, `description`, `stripePriceId`, `price`, `roleId`, `active`, `createdAt`, `updatedAt`), com `price` **em reais (float)**. Pode ser `null` (defensivo, se o plano não existir mais).
- `user` — apenas `id`, `name`, `email` e `role` (`email` é `""` quando `NULL` no banco). **Nunca expõe `passwordHash`** nem outros campos sensíveis. Pode ser `null` (defensivo; não deve ocorrer por causa da FK).
- `invoices` — últimas **20** faturas da assinatura Stripe, ordenadas por `createdAt` **DESC**. Lista vazia → `[]` (nunca `null`/ausente).
  - `amountPaid` é emitido **em reais (float)**; `NULL` no banco (BIGINT) → `0` (nunca `null`).
  - `paidAt` → `null` quando `NULL` no banco; `currency` → `""` quando `NULL`.

**Response `200 OK`:**
```json
{
  "message": "Subscription details retrieved successfully",
  "data": {
    "subscription": {
      "id": "9b2c3d4e-5f6a-4b7c-8d9e-000000000001",
      "userEmail": "joao@mail.com",
      "roleName": "Pro",
      "plan": "Plano Pro",
      "stripeCustomerId": "cus_abc123",
      "stripeSubscriptionId": "sub_xyz789",
      "stripeCheckoutSessionId": "cs_test_abc123",
      "stripePriceId": "price_abc123",
      "status": "active",
      "currentPeriodStart": "2026-01-02T15:04:05Z",
      "currentPeriodEnd": "2026-02-02T15:04:05Z",
      "cancelAtPeriodEnd": false,
      "canceledAt": null,
      "createdAt": "2026-01-02T15:04:05Z",
      "updatedAt": "2026-01-02T15:04:05Z"
    },
    "plan": {
      "id": "3f6c1b1e-0a1b-4c2d-8e3f-000000000001",
      "name": "Plano Pro",
      "description": "Acesso completo",
      "stripePriceId": "price_abc123",
      "price": 49.9,
      "roleId": "a2238759-645f-4d7e-867e-1ccf3d86afef",
      "active": true,
      "createdAt": "2026-01-02T15:04:05Z",
      "updatedAt": "2026-01-02T15:04:05Z"
    },
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "João",
      "email": "joao@mail.com",
      "role": "Standard"
    },
    "invoices": [
      {
        "id": "inv_123",
        "amountPaid": 49.9,
        "currency": "brl",
        "status": "paid",
        "paidAt": "2026-01-02T15:04:05Z",
        "createdAt": "2026-01-02T15:04:05Z"
      }
    ]
  }
}
```

**Sem faturas:**
```json
{
  "message": "Subscription details retrieved successfully",
  "data": {
    "subscription": { "...": "..." },
    "plan": { "...": "..." },
    "user": { "...": "..." },
    "invoices": []
  }
}
```

**Response `400 Bad Request`** — `subscriptionId` não é um UUID válido:
```json
{ "error": "invalid subscription id" }
```

**Response `404 Not Found`** — assinatura não existe:
```json
{ "error": "subscription not found" }
```

#### GET `/admin/users/:userId/subscriptions`

Histórico de assinaturas de um usuário, ordenado por `createdAt` **DESC**. **Somente leitura**.

- `user` — apenas `id`, `name`, `email`, `role` (nunca campos sensíveis).
- `subscriptions` — mesmo shape da lista de `GET /admin/subscriptions` (15 campos). Lista vazia → `[]` (nunca `null`/ausente).

**Response `200 OK`:**
```json
{
  "message": "User subscriptions retrieved successfully",
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "João",
      "email": "joao@mail.com",
      "role": "Standard"
    },
    "subscriptions": [
      {
        "id": "9b2c3d4e-5f6a-4b7c-8d9e-000000000001",
        "userEmail": "joao@mail.com",
        "roleName": "Pro",
        "plan": "Plano Pro",
        "stripeCustomerId": "cus_abc123",
        "stripeSubscriptionId": "sub_xyz789",
        "stripeCheckoutSessionId": "cs_test_abc123",
        "stripePriceId": "price_abc123",
        "status": "active",
        "currentPeriodStart": "2026-01-02T15:04:05Z",
        "currentPeriodEnd": "2026-02-02T15:04:05Z",
        "cancelAtPeriodEnd": false,
        "canceledAt": null,
        "createdAt": "2026-01-02T15:04:05Z",
        "updatedAt": "2026-01-02T15:04:05Z"
      }
    ]
  }
}
```

**Sem assinaturas:**
```json
{
  "message": "User subscriptions retrieved successfully",
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "João",
      "email": "joao@mail.com",
      "role": "Standard"
    },
    "subscriptions": []
  }
}
```

**Response `400 Bad Request`** — `userId` não é um UUID válido:
```json
{ "error": "invalid user id" }
```

**Response `404 Not Found`** — usuário não existe:
```json
{ "error": "user not found" }
```

> Os dois endpoints também retornam `401` sem token (ou token inválido/expirado),
> `403` sem a role `Admin` e `500` `{"error": "..."}` em erro interno.

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
| `422` | Unprocessable Entity — plano sem preço Stripe configurado |
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
| `STRIPE_SECRET_KEY` | - | Chave secreta da API Stripe (fallback: `STRIPE_API_KEY`) |
| `STRIPE_WEBHOOK_SECRET` | - | Segredo do endpoint de webhook Stripe (`whsec_...`), usado para validar `Stripe-Signature` |
| `STRIPE_PRICE_ID` | - | Price ID consumido pelo **frontend** no checkout (o backend não lê esta variável) |
| `SCRAPPER_URL` | - | URL do serviço externo de scraping |
