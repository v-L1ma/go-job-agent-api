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
| POST | `/login/google` | Autenticar com Google (ID Token) |
| POST | `/register` | Criar nova conta (envia e-mail de ativação) |
| POST | `/verify-email` | Confirmar e-mail com o token recebido por e-mail |
| POST | `/resend-confirmation` | Reenviar o link de ativação da conta |
| POST | `/refresh-token` | Renovar token JWT |
| POST | `/forgot-password` | Solicitar link de recuperação de senha (enviado por e-mail) |
| POST | `/reset-password` | Executar reset de senha com o token do e-mail |
| GET | `/plans` | Listar planos de assinatura |
| POST | `/webhook/stripe` | Webhook do Stripe (validado por `Stripe-Signature`) |
| POST | `/cv-analysis/public` | Análise de currículo por IA (score + melhorias) — teste público, sem persistência, rate limit de 5/h por IP |
| POST | `/cv-analysis/public/improved-cv` | PDF do currículo melhorado a partir da análise pública — sem persistência, rate limit próprio de 5/h por IP |

### Privadas (requerem JWT)

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/jobs` | Listar vagas (paginação por cursor + filtros) |
| GET | `/jobs/:jobId` | Detalhes de uma vaga |
| POST | `/jobs/:jobId/rate` | Avaliar uma vaga |
| POST | `/jobs/:jobId/cv` | Gerar currículo personalizado para a vaga (`429` ao atingir o limite diário do plano) |
| GET | `/questions` | Listar perguntas respondidas pelo usuário |
| POST | `/questions` | Responder perguntas de uma vaga (IA) |
| PUT | `/questions/:questionId` | Editar a resposta de uma pergunta |
| POST | `/applications` | Candidatar-se a uma vaga (`429` ao atingir o limite diário do plano) |
| GET | `/applications` | Listar candidaturas (com perguntas) |
| POST | `/applications/:applicationId/rate` | Avaliar uma candidatura |
| POST | `/cv` | Upload de currículo (PDF) |
| GET | `/cv` | Baixar currículo do usuário (PDF) |
| GET | `/cv/generated` | Listar currículos gerados |
| GET | `/cv/:cvId` | Baixar currículo gerado por ID (PDF) |
| POST | `/cv-analysis` | Analisar currículo com IA (`429` ao atingir o limite diário do plano) — persiste no histórico |
| POST | `/cv-analysis/improved-cv` | PDF do currículo melhorado a partir de uma análise já obtida (sem persistência) |
| GET | `/cv-analysis` | Histórico de análises do usuário (máx. 50, `createdAt` desc) |
| GET | `/cv-analysis/:analysisId` | Detalhe de uma análise do usuário (`404` se não existir ou for de outro usuário) |
| GET | `/profile` | Obter perfil do usuário |
| PUT | `/profile` | Atualizar perfil |
| POST | `/preferences` | Criar/atualizar preferências |
| GET | `/preferences` | Obter preferências |
| PUT | `/change-password` | Alterar senha |
| GET | `/statistics` | Obter estatísticas do dashboard |
| POST | `/checkout/session/:planId` | Criar sessão de checkout Stripe (client secret) |
| GET | `/subscriptions/me` | Obter assinatura do usuário autenticado |
| POST | `/subscriptions/me/cancel` | Agendar cancelamento ao fim do período |
| POST | `/subscriptions/me/resume` | Reverter cancelamento agendado |
| POST | `/subscriptions/me/portal-session` | Criar sessão do Billing Portal (Stripe) |
| GET | `/subscriptions/me/invoices` | Listar faturas da assinatura do usuário |

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
| PUT | `/admin/plans/:planId/limits` | Atualizar os limites diários de um plano (currículos e candidaturas) |
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

**Response `403 Forbidden` (e-mail ainda não confirmado):**
```json
{
  "error": "E-mail não confirmado. Verifique sua caixa de entrada ou reenvie o link de ativação."
}
```

> Após validar a senha, se `"EmailConfirmed" = false` o login é bloqueado com
> `403` **antes** de emitir o token JWT. O usuário deve confirmar o e-mail pelo
> link enviado no `/register` (ou reenviar com `/resend-confirmation`).

---

### POST `/login/google`

Autentica (ou cria) o usuário a partir de um **ID Token do Google**. O token é
validado contra o audience configurado em `GOOGLE_CLIENT_ID`.

**Request:**
```json
{
  "idToken": "eyJhbGciOiJSUzI1NiIs..."
}
```

**Response `200 OK` (mesmo shape do `/login`):**
```json
{
  "message": "Login efetuado com sucesso!",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

> Se o e-mail ainda não existir, a conta é criada automaticamente com a role
> `Standard`.

**Response `400 Bad Request`:**
```json
{
  "error": "idToken é obrigatório"
}
```

**Response `401 Unauthorized`:**
```json
{
  "error": "Google ID Token inválido"
}
```

---

### POST `/register`

Cria uma nova conta de usuário e envia o **e-mail de ativação**.

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
  "Message": "Usuário criado com sucesso. Verifique seu e-mail para ativar sua conta."
}
```

> **Confirmação de e-mail:** a conta é criada com `"EmailConfirmed": false` e
> só permite login depois de `POST /verify-email`.
>
> Um token de 32 bytes é gerado e guardado **apenas como hash SHA-256** em
> `EmailConfirmationTokens` (expira em **24 horas**, single-use). O link enviado
> por e-mail é `{FRONTEND_URL}/verify-email?token={token}`.
>
> O e-mail usa o template Resend `account-confirmation`. **Falha no envio não
> falha o registro**: o erro é apenas logado e a resposta continua `201` (o
> usuário pode reenviar com `/resend-confirmation`).
>
> Usuário + role + token são criados em **uma única transação**.

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

Solicita a redefinição de senha. O link de recuperação é enviado **por
e-mail** para o endereço informado.

**Request:**
```json
{
  "email": "usuario@email.com"
}
```

**Response `200 OK` (sempre — email existe ou não):**
```json
{
  "message": "Se o e-mail existir, você receberá um link de recuperação."
}
```

> **Segurança:** a resposta é **idêntica** nos dois casos (e-mail cadastrado
> ou não) para não revelar a existência da conta. O token **nunca** aparece
> na resposta nem em log de request/response.
>
> O token é guardado apenas como hash SHA-256 em `PasswordResetTokens`,
> expira em **1 hora** e é **single-use**. O link enviado é
> `{FRONTEND_URL}/reset-password?token={token}`.
>
> O e-mail usa o template Resend `password-reset` via
> `Emailer.SendWithTemplate(...)` (ver "Serviço de E-mail").
>
> Falha no envio do e-mail também devolve `200` (apenas loga o erro no
> servidor) — o usuário jamais deve distinguir os casos.

**Variáveis de ambiente do envio (ver "Serviço de E-mail" e a tabela
completa no final):** `RESEND_API_KEY`, `FRONTEND_URL`.

---

### POST `/reset-password`

Redefine a senha usando o token recebido **por e-mail** no fluxo de
`/forgot-password` (o frontend lê o query param `?token=` da URL
`{FRONTEND_URL}/reset-password?token=...` e envia aqui).

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

> Trocar a senha e consumir o token acontecem **na mesma transação**: ou as
> duas coisas acontecem, ou nenhuma (token single-use + expiração de 1h).

**Response `200 OK`:**
```json
{
  "message": "Senha redefinida com sucesso!"
}
```

**Response `400 Bad Request`** (token inválido, já usado ou expirado):
```json
{
  "error": "Token inválido ou expirado."
}
```

**Response `400 Bad Request`** (senhas não conferem):
```json
{
  "error": "As senhas não coincidem."
}
```

---

### POST `/verify-email`

Confirma a conta a partir do token recebido por e-mail no `/register` (ou no
`/resend-confirmation`). Rota **pública** (sem JWT). O frontend lê o query
param `?token=` da URL `{FRONTEND_URL}/verify-email?token=...` e envia aqui.

**Request:**
```json
{
  "token": "a1b2c3d4e5f6..."
}
```

Validações:
- `token`: obrigatório

**Response `200 OK`:**
```json
{
  "message": "E-mail confirmado com sucesso!"
}
```

> **Idempotente:** se a conta **já está confirmada**, a resposta também é `200`
> com a mesma mensagem — independentemente de o token já ter sido usado ou
> ter expirado. Isso existe porque scanners de e-mail abrem os links antes do
> usuário.
>
> O token é buscado **sem** filtro de `Used`/`ExpiresAt`; a decisão é tomada
> pelo servidor: token inexistente → `400`; conta não confirmada + token
> expirado/já usado → `400`; caso válido → `UPDATE "AspNetUsers"
> SET "EmailConfirmed" = true` + marca o token como usado **na mesma
> transação** (single-use).

**Response `400 Bad Request`** (token inválido, desconhecido, expirado ou já
usado com conta ainda não confirmada):
```json
{
  "error": "Token inválido ou expirado."
}
```

---

### POST `/resend-confirmation`

Reenvia o link de ativação da conta. Rota **pública** (sem JWT).

**Request:**
```json
{
  "email": "joao@email.com"
}
```

Validações:
- `email`: obrigatório, formato email

**Response `200 OK` (sempre — e-mail existe ou não, confirmado ou não):**
```json
{
  "message": "Se o e-mail existir e ainda não estiver confirmado, você receberá um novo link de ativação."
}
```

> **Segurança:** a resposta é **idêntica** em todos os casos para não revelar
> a existência da conta.
>
> O e-mail só é enviado quando o usuário existe **e** `"EmailConfirmed" =
> false`. A cada chamada é criado um **novo** token de 24h (hash SHA-256 em
> `EmailConfirmationTokens`); os anteriores continuam válidos até expirarem.
>
> Usa o mesmo template Resend `account-confirmation` e o mesmo link
> `{FRONTEND_URL}/verify-email?token={token}` do `/register`. Falha no envio
> (ou erro interno) é apenas logada — a resposta continua `200`.

---

### POST `/cv-analysis/public`

Avalia um currículo (PDF) com IA e devolve um **score de 0 a 100**, parecer,
categorias, pontos fortes e melhorias acionáveis.

Rota **pública** (sem JWT), feita para teste de visitantes:

- **não persiste nada** (nenhum registro é criado);
- aplica **rate limit de 5 requisições por hora por IP** (janela deslizante,
  identificação via `c.RealIP()`);
- **não** passa pelo limite diário do plano (isso vale só para a rota
  privada `POST /cv-analysis`).

**Request:** `multipart/form-data`

| Campo | Tipo | Obrigatório | Restrições |
|-------|------|-------------|-----------|
| `cv` | file | Sim | Apenas `application/pdf`, até **10MB** |
| `jobTitle` | string | Não\* | Máximo de **200** caracteres |
| `jobDescription` | string | Não\* | Máximo de **10000** caracteres |

\* `jobTitle` e `jobDescription` andam **juntos**: enviado um deles, o outro
torna-se obrigatório. Sem vaga informada, o objeto `job` vem `null`; com
vaga, as melhorias passam a considerar o contexto da vaga.

**Response `200 OK`:**
```json
{
  "data": {
    "score": 78,
    "summary": "Currículo bem estruturado, com resultados claros, mas sem detalhes de liderança.",
    "categories": [
      { "name": "Estrutura e formatação", "score": 90, "comment": "Layout limpo e fácil de ler." },
      { "name": "Conteúdo e impacto", "score": 75, "comment": "Resultados quantitativos presentes." },
      { "name": "Clareza e objetividade", "score": 95, "comment": "Texto direto e objetivo." },
      { "name": "Palavras-chave", "score": 70, "comment": "Faltam termos específicos da vaga." }
    ],
    "strengths": [
      { "area": "experiencias", "text": "Experiência sólida com Go e Kubernetes" },
      { "area": "experiencias", "text": "Métricas quantitativas nas experiências" }
    ],
    "improvements": [
      {
        "area": "experiencias",
        "severity": "alta",
        "problem": "As descrições focam em implementação, sem mencionar decisões técnicas.",
        "suggestion": "Descrever participações no desenho da arquitetura e decisões tomadas."
      },
      {
        "area": "skills",
        "severity": "media",
        "problem": "Ferramentas de observabilidade não estão na lista de skills.",
        "suggestion": "Incluir Prometheus/Grafana caso tenha experiência."
      }
    ],
    "job": null,
    "cv": {
      "Nome": "João da Silva",
      "Email": "joao@email.com",
      "Telefone": "(11) 99999-9999",
      "Linkedin": "https://linkedin.com/in/joao",
      "Github": "https://github.com/joao",
      "Resumo": "Desenvolvedor backend com 5 anos de experiência...",
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
      ],
      "SecoesExtras": [
        { "Titulo": "Idiomas", "Conteudo": "Inglês — Avançado\nEspanhol — Básico" },
        { "Titulo": "Certificações", "Conteudo": "AWS Certified Solutions Architect — 2024" }
      ]
    }
  }
}
```

> Com vaga informada, `"job"` deixa de ser `null` e passa a:
> ```json
> {
>   "title": "Dev Backend Pleno Go",
>   "matchScore": 65,
>   "missingSkills": ["AWS", "RabbitMQ"],
>   "comment": "Base técnica adequada, mas faltam requisitos da vaga."
> }
> ```

**Objeto `analysis`** (contrato comum às rotas pública e privada):

| Campo | Tipo | Regras |
|-------|------|--------|
| `score` | int | `0`–`100` (clampado) |
| `summary` | string | Parecer geral em 1 a 3 frases |
| `categories` | array | **4 a 6** entradas `{ name, score (0–100), comment }` |
| `strengths` | array | **2 a 5** entradas `{ area, text }` (sempre array, nunca `null`) |
| `strengths[].area` | string | `estrutura` \| `contato` \| `resumo` \| `skills` \| `experiencias` \| `educacao` \| `geral` — seção do documento em que o viewer ancora/highlighta o ponto forte |
| `improvements` | array | **3 a 8** entradas `{ area, severity, problem, suggestion }` |
| `improvements[].area` | string | `estrutura` \| `contato` \| `resumo` \| `skills` \| `experiencias` \| `educacao` \| `geral` |
| `improvements[].severity` | string | `alta` \| `media` \| `baixa` |
| `job` | object \| null | `null` quando nenhuma vaga foi enviada; com vaga: `{ title, matchScore (0–100), missingSkills: string[], comment }` |
| `cv` | object | Currículo estruturado no **mesmo shape PascalCase** de `GeneratedCv` (`Nome`, `Email`, `Telefone`, `Linkedin`, `Github`, `Resumo`, `Skills`, `Experiencias`, `Educacao`, `SecoesExtras`) — pronta para o viewer do frontend |
| `cv.SecoesExtras` | array | Seções do documento que **não cabem nos campos fixos** (Idiomas, Certificações, Projetos, Cursos, Prêmios, Objetivos, Informações adicionais etc.): `{ Titulo, Conteudo }`, com `Titulo` = nome da seção como no documento e `Conteudo` = conteúdo integral (quebras de linha preservadas). Array vazio quando não há — **nunca `null`** |

> **Transcrição completa:** o `cv` é uma transcrição **verbatim** do PDF — o
> backend instrui explicitamente a IA a não resumir, abreviar, cortar,
> parafrasear ou omitir nada (texto de `Resumo`/`Descricao` original, datas,
> nomes de empresa/instituição e links completos). O que não couber nos campos
> fixos vai para `SecoesExtras`, então nenhuma seção do documento se perde.
> Compatível retro: análises antigas persistidas **sem** `SecoesExtras`
> devolvem `[]`.

> **Datas ignoradas em relação ao "hoje":** a análise **não avalia nem
> pontua** anos "no futuro", datas "inconsistentes com a realidade atual" ou
> cronologia frente à data de hoje (a IA não tem acesso à data atual), e
> `improvements` exclusivamente sobre isso são **descartados na
> normalização** — também em análises já persistidas, lidas via
> `GET /cv-analysis/:analysisId`. Contradições **internas** de datas (fim
> antes do início, mês impossível, períodos que se contradizem) continuam
> sendo apontadas normalmente. O currículo melhorado também preserva as datas
> exatamente como estão, sem "corrigir" anos.

Todos os campos acima sempre vêm presentes (listas vazias em vez de `null`),
mesmo quando a IA devolve menos entradas do que o mínimo — itens nunca são
inventados.

> **Compatibilidade:** análises antigas persistidas com `strengths` como
> array de strings puras (formato legado `["forte1", "forte2"]`) são
> **coeridas na leitura** para o formato atual: cada string vira
> `{ "area": ..., "text": <string> }`, com `area` inferida por palavra-chave
> do texto (ex.: "experiência" → `experiencias`) e fallback
> `"geral"` quando não há match.

**Response `400 Bad Request`:**
```json
{ "error": "Currículo não enviado: envie o arquivo PDF no campo \"cv\"." }
```
```json
{ "error": "Tipo de arquivo inválido: envie um PDF (application/pdf)." }
```
```json
{ "error": "Os campos jobTitle e jobDescription devem ser enviados juntos: informe o título e a descrição da vaga." }
```
```json
{ "error": "O campo jobTitle deve ter no máximo 200 caracteres." }
```
```json
{ "error": "O campo jobDescription deve ter no máximo 10000 caracteres." }
```
```json
{ "error": "Erro ao abrir o PDF: envie um arquivo PDF válido." }
```

**Response `413 Payload Too Large`:**
```json
{ "error": "O arquivo excede o tamanho máximo permitido de 10MB." }
```
```json
{ "error": "A requisição excede o tamanho máximo permitido de 10MB." }
```

**Response `429 Too Many Requests` (rate limit por IP):**
```json
{ "error": "Muitas tentativas. Tente novamente mais tarde." }
```

**Response `500 Internal Server Error`:**
```json
{ "error": "Erro ao gerar análise do currículo" }
```
```json
{ "error": "Erro ao analisar resposta da IA: resposta da IA não é um JSON válido: ..." }
```

---

### POST `/cv-analysis/public/improved-cv`

Gera o **PDF de um currículo melhorado** a partir de uma análise que o cliente
já possui — no fluxo público, a análise que veio do `POST /cv-analysis/public`.
A IA reescreve o currículo aplicando as dicas de `improvements`, **sem inventar
nada e sem cortar conteúdo**, e o resultado volta como download de PDF.

Rota **pública** (sem JWT), com **rate limit próprio de 5 requisições por hora
por IP** (limitador separado do da análise: esgotar um não bloqueia o outro).
**Não persiste nada** e não passa pelo limite diário do plano.

**Request:** `application/json`

| Campo | Tipo | Obrigatório | Restrições |
|-------|------|-------------|-----------|
| `analysis` | object | Sim | Objeto `analysis` **completo** e no mesmo contrato da rota pública (o recebido no `data` da resposta do `POST /cv-analysis/public`) |
| `analysis.cv` | object | Sim | Currículo estruturado (`cv`) da análise — é ele que vira o PDF. Objeto vazio (`{}`) também é rejeitado |

```json
{
  "analysis": {
    "score": 78,
    "summary": "Currículo bem estruturado, mas sem detalhes de liderança.",
    "categories": [
      { "name": "Conteúdo e impacto", "score": 70, "comment": "Faltam resultados quantitativos." }
    ],
    "strengths": [
      { "area": "experiencias", "text": "Experiência sólida com Go e Kubernetes" }
    ],
    "improvements": [
      {
        "area": "experiencias",
        "severity": "alta",
        "problem": "As descrições focam em implementação, sem mencionar decisões técnicas.",
        "suggestion": "Descrever participações no desenho da arquitetura e decisões tomadas."
      },
      {
        "area": "skills",
        "severity": "media",
        "problem": "Ferramentas de observabilidade não estão na lista de skills.",
        "suggestion": "Incluir Prometheus/Grafana caso tenha experiência."
      }
    ],
    "job": null,
    "cv": {
      "Nome": "João da Silva",
      "Email": "joao@email.com",
      "Telefone": "(11) 99999-9999",
      "Linkedin": "https://linkedin.com/in/joao",
      "Github": "https://github.com/joao",
      "Resumo": "Desenvolvedor backend com 5 anos de experiência...",
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
      ],
      "SecoesExtras": [
        { "Titulo": "Idiomas", "Conteudo": "Inglês — Avançado\nEspanhol — Básico" }
      ]
    }
  }
}
```

Os campos `summary`, `categories` e `job` (quando presentes) entram no prompt
apenas como **contexto**; `strengths` e `score` não afetam o resultado. O que
volta ao cliente é somente o `cv` reescrito, aplicando as sugestões de
`improvements`.

**Response `200 OK`:** PDF binário.

```
Content-Type: application/pdf
Content-Disposition: attachment; filename="curriculo-melhorado.pdf"
```

O corpo é o arquivo PDF gerado (mesmo motor das rotas `POST /jobs/:jobId/cv`,
`GET /cv` e `GET /cv/:cvId`), com as seções de `cv.SecoesExtras` renderizadas
após a Educação.

**Response `400 Bad Request`:**
```json
{ "error": "Corpo da requisição vazio: envie um JSON no formato {\"analysis\": {...}}." }
```
```json
{ "error": "Corpo da requisição inválido: envie um JSON no formato {\"analysis\": {...}}." }
```
```json
{ "error": "Campo \"analysis\" ausente: envie a análise completa retornada por POST /cv-analysis (ou pela rota pública)." }
```
```json
{ "error": "Campo \"analysis\" inválido: esperado um objeto JSON com a análise." }
```
```json
{ "error": "Campo \"analysis.cv\" ausente: a análise precisa conter o currículo estruturado para gerar o PDF." }
```
```json
{ "error": "Campo \"analysis.cv\" veio vazio: envie a análise com o currículo preenchido." }
```

**Response `413 Payload Too Large`:**
```json
{ "error": "A requisição excede o tamanho máximo permitido de 1MB." }
```

**Response `429 Too Many Requests` (rate limit por IP):**
```json
{ "error": "Muitas tentativas. Tente novamente mais tarde." }
```

**Response `500 Internal Server Error`:**
```json
{ "error": "Erro ao gerar o currículo melhorado" }
```
```json
{ "error": "Erro ao analisar resposta da IA: resposta da IA não é um JSON válido: ..." }
```
```json
{ "error": "A IA não devolveu um currículo válido para gerar o PDF" }
```
```json
{ "error": "Erro ao gerar o PDF do currículo melhorado" }
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
| `cursor` | string (base64) | não | - | Cursor devolvido em `nextCursor` (página seguinte) |
| `platform` | string | não | - | Filtra por plataforma (**igualdade exata**, ex.: `LinkedIn`) |
| `stack` | string | não | - | Filtra por **cargo/título** da vaga (`Title ILIKE`, case-insensitive) — rótulo da UI: "Cargo ou Stack" |
| `company` | string | não | - | Filtra por **empresa** (`Company ILIKE`, case-insensitive) |

> **`location` NÃO é suportado.** A tabela `Jobs` não possui coluna de
> localização, então o filtro não existe no backend — enviá-lo é silenciosamente
> ignorado. A UI não deve exibir/enviar esse parâmetro.

> Os filtros `platform`, `stack` e `company` são aplicados **antes** da
> comparação do cursor (mesma CTE), então a paginação por cursor continua
> estável mesmo alternando filtros entre as páginas.

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
      "company": "Empresa XYZ",
      "score": 0.87
    }
  ],
  "nextCursor": "eyJzIjowLjg3LCJjIjoiMjAyNi0wNy0wM1QxMDowMDowMFoiLCJpIjoiNTUwZTg0MDAt..."
}
```

Para obter a próxima página, envie o valor exato de `nextCursor` como
`?cursor=<nextCursor>` (é um base64 de `{s: score, c: createdAt, i: jobId}`).
`nextCursor` ausente/`""` significa última página.

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

> O usuário deve ter feito upload de um currículo via `POST /cv` antes de usar esta rota.

**Response `429 Too Many Requests`** — limite diário do plano atingido
(ver [Limites diários por plano](#limites-diários-por-plano)):
```json
{
  "error": "Você atingiu o limite diário de 5 currículos do seu plano. Tente novamente amanhã."
}
```

> A verificação acontece **antes** da chamada à IA: requisições que excedem o
> limite não consomem cota do Gemini. O limite é diário em UTC; `null` no
> plano = ilimitado; `0` = sempre bloqueado. Usuários sem assinatura ativa
> não têm limite.

---

### GET `/questions`

Lista as perguntas de formulário já respondidas pelo usuário (e a vaga
relacionada).

**Response `200 OK`:**
```json
{
  "message": "Perguntas encontradas com sucesso!",
  "data": [
    {
      "Question": "Qual sua pretensão salarial?",
      "Answer": "R$ 12.000",
      "Title": "Desenvolvedor Go"
    }
  ]
}
```

> As chaves seguem o nome da coluna (PascalCase) — a linha vem direto do
> sqlc. `Title` é `null` quando a vaga relacionada não existe mais.
> Sem perguntas, o campo `data` é **omitido** (envelope com `omitempty`).

---

### POST `/questions`

Recebe as perguntas de um formulário de candidatura (`jobId`), responde as
que ainda não foram respondidas usando IA (Gemini) a partir do currículo +
descrição da vaga, grava as respostas em transação e devolve **todas** as
respostas (as já existentes + as novas).

**Request:**
```json
{
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "questoes": [
    {
      "pergunta": "Você tem experiência com Go?",
      "tipo": "text",
      "opcoes": ["Sim", "Não"]
    },
    {
      "pergunta": "Qual sua pretensão salarial?",
      "tipo": "numeric"
    }
  ]
}
```

Validações/campos:
- `jobId`: obrigatório (UUID)
- `questoes[].pergunta`: obrigatório
- `questoes[].tipo`: tipo da pergunta (ex.: `text`, `numeric`, `select`)
- `questoes[].opcoes`: opcional (opcções quando o tipo é de escolha)

**Response `200 OK`:**
```json
{
  "message": "Respostas recebidas com sucesso!",
  "data": [
    {
      "pergunta": "Você tem experiência com Go?",
      "resposta": "Sim, atuo com Go há 4 anos..."
    },
    {
      "pergunta": "Qual sua pretensão salarial?",
      "resposta": "12000"
    }
  ]
}
```

> Se **todas** as perguntas já tiverem resposta no banco, a IA não é chamada:
> as respostas existentes são devolvidas no mesmo shape.

**Response `500 Internal Server Error`:**
```json
{
  "error": "Resposta inválida da IA"
}
```

---

### PUT `/questions/:questionId`

Edita a resposta de uma pergunta.

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `questionId` | UUID | ID da pergunta |

**Request:**
```json
{
  "answer": "Sim, tenho 5 anos de experiência com Go e PostgreSQL."
}
```

Validações:
- `answer`: obrigatório, min 2, max 2000

**Response `200 OK`:**
```json
{
  "Message": "Perguntas encontradas com sucesso!"
}
```

> Atenção: esta rota devolve a chave **`Message`** (maiúscula), e não
> `message` — é o contrato atual do handler.

**Response `400 Bad Request` (validação):**
```json
{
  "error": "Key: 'EditQuestion.Answer' Error:Field validation for 'Answer' failed on the 'min' tag"
}
```

**Response `500 Internal Server Error`:**
```json
{
  "error": "Questão não pertence a outro usuário."
}
```
(retornado quando a pergunta pertence a outro usuário, ou quando o UUID é
inválido)

---

### POST `/applications`

Candidata-se a uma vaga. O `jobId` vai no **body** (não há rota
`/jobs/:jobId/apply`).

**Request:**
```json
{
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "applied",
  "observation": "Candidatura enviada pelo painel",
  "platform": "LinkedIn"
}
```

**Response `201 Created`:**
```json
{
  "message": "Candidatura concluída com sucesso!"
}
```

**Response `400 Bad Request`:**
```json
{
  "error": "Invalid request body"
}
```

**Response `429 Too Many Requests`** — limite diário do plano atingido
(ver [Limites diários por plano](#limites-diários-por-plano)):
```json
{
  "error": "Você atingiu o limite diário de 10 candidaturas do seu plano. Tente novamente amanhã."
}
```

> A verificação acontece antes do INSERT da candidatura. O limite é diário em
> UTC; `null` no plano = ilimitado; `0` = sempre bloqueado. Usuários sem
> assinatura ativa não têm limite.

---

### GET `/applications`

Lista as candidaturas do usuário (com `status != closed`), mais recentes
primeiro, incluindo as perguntas já respondidas para cada vaga.

**Response `200 OK`:**
```json
{
  "message": "Candidaturas encontradas com sucesso!",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "title": "Desenvolvedor Go",
      "company": "Empresa XYZ",
      "platform": "LinkedIn",
      "status": "applied",
      "createdAt": "2026-07-03T10:00:00Z",
      "questionsCount": 3,
      "questions": [
        {
          "id": "550e8400-e29b-41d4-a716-446655440002",
          "question": "Qual sua pretensão salarial?",
          "answer": "R$ 12.000",
          "type": "text",
          "options": []
        }
      ]
    }
  ]
}
```

> Quando não há candidaturas, o campo `data` é **omitido** (o envelope usa
> `omitempty`) — trate `data` como opcional no frontend.

---

### POST `/applications/:applicationId/rate`

Avalia uma candidatura (curtir/não curtir).

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `applicationId` | UUID | ID da candidatura |

**Request:**
```json
{
  "liked": true,
  "feedback": "Processo rápido e transparente"
}
```

Validações:
- `liked`: obrigatório (bool)
- `feedback`: opcional; min 3, max 600

**Response `201 Created`:**
```json
{
  "message": "Candidatura avaliada com sucesso!"
}
```

**Response `400 Bad Request` (validação):**
```json
{
  "errors": {
    "RateApplicationRequest.feedback": "Este campo deve ter no mínimo 3 caracteres"
  }
}
```

---

### POST `/cv`

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
    ],
    "SecoesExtras": null
  }
}
```

> `response` é o shape completo `GeneratedCv`, que inclui `SecoesExtras`
> (Idiomas, Certificações, Projetos etc.). Nesta rota o prompt de estruturação
> **não** pede essas seções, então o campo vem `null` — trate-o como opcional.
> Já nas rotas de análise ele vem sempre como array (`[]` quando vazio).

**Response `400 Bad Request`:**
```json
{
  "error": "Currículo não enviado"
}
```

---

### GET `/cv`

Baixa o currículo do usuário como PDF.

**Response `200 OK`:** PDF binário (`application/pdf`)

**Response `404 Not Found`:**
```json
{
  "error": "O curriculo do usuario informado não foi encontrado"
}
```

---

### GET `/cv/generated`

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

### GET `/cv/:cvId`

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

### GET `/profile`

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

### PUT `/profile`

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

### POST `/preferences`

Cria ou atualiza as preferências de busca de vagas do usuário (skills e níveis de senioridade).

**Request:**
```json
{
  "skills": ["Go", "PostgreSQL", "Docker", "Kubernetes"],
  "levels": ["Junior", "Pleno", "Senior"],
  "precision": 0.7
}
```

Validações:
- `skills`: obrigatório, min 1 item
- `levels`: obrigatório, min 1 item
- `precision`: obrigatório, entre `0.0` e `1.0` (percentual mínimo de
  similaridade usado ao listar vagas)

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

### GET `/preferences`

Retorna as preferências de busca do usuário.

**Response `200 OK`:**
```json
{
  "message": "Preferências encontradas com sucesso!",
  "data": {
    "UserId": "550e8400-e29b-41d4-a716-446655440000",
    "SimilarityPercent": 0.7,
    "Keywords": ["Go", "PostgreSQL", "Docker"],
    "Levels": ["Pleno", "Senior"]
  }
}
```

> `data` é um **objeto** (não lista), com os nomes das colunas em
> PascalCase. Sem preferências, `Keywords`/`Levels` vêm vazios (`[]`) e
> `SimilarityPercent` é `0.7` (padrão).

---

### PUT `/change-password`

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

### GET `/statistics`

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

### POST `/cv-analysis`

Mesma análise da rota pública, mas para o usuário autenticado: aplica o
**limite diário do plano** e **persiste** a análise em `CvAnalyses`
(histórico consultável pelas rotas `GET` abaixo).

A verificação do limite roda **antes** da chamada à IA (não consome cota do
Gemini em requisições que serão rejeitadas). Ver
[Limites diários por plano](#limites-diários-por-plano).

**Headers:**
```
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**Request:** `multipart/form-data` — mesmos campos da rota pública:

| Campo | Tipo | Obrigatório | Restrições |
|-------|------|-------------|-----------|
| `cv` | file | Sim | Apenas `application/pdf`, até **10MB** |
| `jobTitle` | string | Não\* | Máximo de **200** caracteres |
| `jobDescription` | string | Não\* | Máximo de **10000** caracteres |

\* Enviado um, o outro é obrigatório.

**Response `200 OK`:** o objeto `analysis` completo **+** metadados de
persistência (`id` e `createdAt` no mesmo nível dos demais campos):

```json
{
  "data": {
    "id": "79f36725-23c9-4979-b7b1-e8c6fc1984ac",
    "createdAt": "2026-10-14T17:42:48Z",
    "score": 75,
    "summary": "Currículo bem estruturado e objetivo...",
    "categories": [
      { "name": "Estrutura e formatação", "score": 90, "comment": "Layout limpo." }
    ],
    "strengths": [
      { "area": "experiencias", "text": "Experiência sólida com Go e Kubernetes" }
    ],
    "improvements": [
      {
        "area": "experiencias",
        "severity": "alta",
        "problem": "Descrições focadas em implementação.",
        "suggestion": "Detalhar decisões de arquitetura."
      }
    ],
    "job": {
      "title": "Arquiteto de Software",
      "matchScore": 65,
      "missingSkills": ["Liderança técnica", "Observabilidade"],
      "comment": "Base técnica adequada, perfil ainda executando."
    },
    "cv": { "Nome": "João da Silva", "Skills": ["Go", "Kubernetes"] }
  }
}
```

> `createdAt` é RFC3339 em UTC. Sem vaga informada, `"job": null` e a
> resposta é idêntica à da rota pública, apenas com `id` e `createdAt`.

| Código | Quando |
|--------|--------|
| `400` | Arquivo ausente, tipo inválido, PDF ilegível, campos de vaga incompletos/longos demais |
| `401` | Token ausente, inválido ou expirado (middleware JWT) |
| `413` | Arquivo/corpo acima de 10MB |
| `429` | Limite diário do plano atingido (`{"error": "Você atingiu o limite diário de N análises de currículo do seu plano. Tente novamente amanhã."}`) |
| `500` | Falha na IA ou resposta ilegível |

---

### GET `/cv-analysis`

Lista o histórico de análises do usuário autenticado — **no máximo 50
itens**, ordenado por `createdAt` **decrescente**.

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "3abbb821-95d8-4ede-b235-3685ffb0616b",
      "score": 84,
      "summary": "Currículo objetivo e bem estruturado...",
      "jobTitle": null,
      "createdAt": "2026-10-14T17:43:07Z"
    },
    {
      "id": "79f36725-23c9-4979-b7b1-e8c6fc1984ac",
      "score": 75,
      "summary": "Currículo bem estruturado e objetivo...",
      "jobTitle": "Arquiteto de Software",
      "createdAt": "2026-10-14T17:42:48Z"
    }
  ]
}
```

- `jobTitle`: título da vaga quando a análise foi feita **com** vaga, senão `null`.
- Sem análises, responde `{"data": []}` (a chave `data` **nunca** é omitida).

**Response `401 Unauthorized`:** token ausente/inválido/expirado.

---

### GET `/cv-analysis/:analysisId`

Retorna o detalhe de uma análise: `id`, `createdAt` e o objeto `analysis`
completo (mesmo formato retornado pelo `POST`).

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `analysisId` | UUID | ID da análise (obtido no `POST /cv-analysis` ou na lista) |

**Response `200 OK`:**
```json
{
  "data": {
    "id": "79f36725-23c9-4979-b7b1-e8c6fc1984ac",
    "createdAt": "2026-10-14T17:42:48Z",
    "score": 75,
    "summary": "...",
    "categories": [],
    "strengths": [],
    "improvements": [],
    "job": null,
    "cv": { "Nome": "João da Silva" }
  }
}
```

> **Análises antigas:** o `Result` persistido é texto JSON lido e revalidado
> contra o contrato atual na leitura. `strengths` gravado no formato legado
> (`["forte1", "forte2"]`) é coerido para `[{ "area": ..., "text": ... }]`
> — `area` inferida por palavra-chave do texto, com fallback `"geral"` —
> então o cliente sempre recebe o formato `{ area, text }`.

**Response `404 Not Found`:**
```json
{ "error": "Análise não encontrada" }
```

> A busca é filtrada pelo `userId` do JWT: uma análise que **não existe** e
> uma análise que **pertence a outro usuário** retornam exatamente o mesmo
> `404` (sem vazar a existência do recurso). ID com formato inválido também
> responde `404`.

**Response `401 Unauthorized`:** token ausente/inválido/expirado.

---

### POST `/cv-analysis/improved-cv`

Gera o **PDF de um currículo melhorado** a partir de uma análise que o cliente
já possui — normalmente a vinda do histórico (`GET /cv-analysis/:analysisId`)
ou da resposta do `POST /cv-analysis`. A IA reescreve o currículo aplicando as
sugestões de `improvements`, **sem inventar informações** (nada de empregos,
datas, formações ou habilidades fictícias) e **sem resumir nem cortar** o
conteúdo original: o objetivo é melhorar a redação preservando a completude.
Os dados de contato ficam exatamente como estão.

**Não persiste nada** (não cria registro nem consome cota de plano) — só
transforma o corpo enviado em PDF. Espelha a rota pública
`POST /cv-analysis/public/improved-cv`, com a diferença de exigir JWT e de
**não ter rate limit por IP**.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request:** `application/json` — mesmos campos da rota pública:

| Campo | Tipo | Obrigatório | Restrições |
|-------|------|-------------|-----------|
| `analysis` | object | Sim | Objeto `analysis` **completo** no contrato documentado acima (o `data` devolvido pelo `GET /cv-analysis/:analysisId`) |
| `analysis.cv` | object | Sim | Currículo estruturado da análise — é ele que vira o PDF (`{}` é rejeitado) |

Exemplo de corpo: ver o objeto `analysis` do
[POST `/cv-analysis`](#post-cv-analysis) (aceita o mesmo JSON, com ou sem os
metadados `id`/`createdAt` — eles são ignorados).

**Response `200 OK`:** PDF binário.

```
Content-Type: application/pdf
Content-Disposition: attachment; filename="curriculo-melhorado.pdf"
```

| Código | Quando |
|--------|--------|
| `400` | JSON malformado, corpo vazio, `analysis` ausente/`null`, `analysis.cv` ausente ou vazio |
| `401` | Token ausente, inválido ou expirado (middleware JWT) |
| `413` | Corpo acima de **1MB** |
| `500` | Falha na IA, resposta ilegível ou falha ao gerar o PDF |

> **Contrato do PDF:** é o mesmo gerador de `POST /jobs/:jobId/cv` e `GET /cv`
> — cabeçalho com nome + contato, Resumo, Skills, Experiência, Educação e,
> por último, as seções de `cv.SecoesExtras` (título em negrito + conteúdo).

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
- A ativação do acesso **não** acontece aqui: ela ocorre via webhook (`customer.subscription.created/updated`);
- Redirecionamentos pós-pagamento derivam da env `FRONTEND_URL` (fallback `http://localhost:3000`):
  - `return_url` → `{FRONTEND_URL}/checkout/success?session_id={CHECKOUT_SESSION_ID}`
    (`{CHECKOUT_SESSION_ID}` é o placeholder **literal** do Stripe, substituído pelo id real da sessão);
  - `cancel_url` → `{FRONTEND_URL}/checkout/cancel`.

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

### POST `/subscriptions/me/cancel` *(privada)*

Agenda o cancelamento da assinatura para o **fim do período atual**
(`cancelAtPeriodEnd = true`). **Sem body.** É o comportamento exibido pela UI
de "Cancelar assinatura": o acesso continua até `currentPeriodEnd`.

> O dono é sempre resolvido pelo `userId` do JWT — não há id na rota.

**Request:** sem corpo (ou corpo vazio).

**Response `200 OK`** (mesmo shape do `GET /subscriptions/me`):
```json
{
  "message": "Cancelamento agendado para o fim do período.",
  "data": {
    "id": "8f0a1b2c-0000-4000-8000-000000000000",
    "planId": "3f6c1b1e-0a1b-4c2d-8e3f-000000000001",
    "status": "active",
    "cancelAtPeriodEnd": true,
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

> `cancelAtPeriodEnd` é o valor **recém-confirmado pela API do Stripe**
> (a tabela só é sincronizada pelo webhook `customer.subscription.updated`).

**Response `404 Not Found`** — usuário sem assinatura:
```json
{ "error": "subscription not found" }
```

**Response `400 Bad Request`** — assinatura ainda sem id no Stripe (checkout
não concluído):
```json
{ "error": "subscription has no stripe id" }
```

**Response `400 Bad Request` / `401 Unauthorized`** — claim inválido/ausente:
```json
{ "error": "invalid user id" }
```
```json
{ "error": "unauthorized" }
```

**Response `500 Internal Server Error`** — falha na API do Stripe:
```json
{ "error": "failed to update subscription" }
```

---

### POST `/subscriptions/me/resume` *(privada)*

Reverte o cancelamento agendado (`cancelAtPeriodEnd = false`) — o
cancelamento no fim do período é **reversível** enquanto não ocorrer.
**Sem body.** Mesmos caminhos de erro e mesmo shape de resposta do
`/cancel`.

**Response `200 OK`:**
```json
{
  "message": "Cancelamento removido.",
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

Erros: `404` (`subscription not found`), `400`
(`subscription has no stripe id`), `401`/`400` (claim), `500`
(`failed to update subscription`) — idênticos ao `/cancel`.

---

### POST `/subscriptions/me/portal-session` *(privada)*

Cria uma sessão do **Billing Portal** do Stripe (o usuário gerencia cartão,
plano e cancelamento na UI do Stripe). **Sem body.**

**Response `200 OK`:**
```json
{
  "message": "Sessão do portal de cobrança criada com sucesso.",
  "data": {
    "url": "https://billing.stripe.com/session/cs_test_a1b2c3d4..."
  }
}
```

> `return_url` da sessão = `{FRONTEND_URL}/perfil` (env `FRONTEND_URL`).

**Response `404 Not Found`** — usuário sem assinatura:
```json
{ "error": "subscription not found" }
```

**Response `400 Bad Request`** — nenhum Stripe Customer vinculado:
```json
{ "error": "subscription has no stripe customer" }
```

**Response `500 Internal Server Error`:**
```json
{ "error": "failed to create billing portal session" }
```

---

### GET `/subscriptions/me/invoices` *(privada)*

Lista as faturas da assinatura do usuário (tabela `StripeInvoices`), mais
recentes primeiro, no máximo **20**.

**Response `200 OK` (com faturas):**
```json
{
  "message": "Faturas listadas com sucesso.",
  "data": [
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
```

- `amountPaid` é emitido **em reais (float)**; `NULL` no banco → `0`;
- `paidAt` é `null` quando não pago; `currency` é `""` quando `NULL`.

**Response `200 OK` (sem faturas / sem assinatura / checkout não concluído)** —
`data` é **sempre** lista, `[]` nunca `null`:
```json
{
  "message": "Faturas listadas com sucesso.",
  "data": []
}
```

**Response `401 Unauthorized` / `400 Bad Request`** — claim ausente/inválido:
```json
{ "error": "unauthorized" }
```
```json
{ "error": "invalid user id" }
```

**Response `500 Internal Server Error`:**
```json
{ "error": "..." }
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

### Limites diários por plano

Cada plano (`Plans`) pode definir dois limites diários, aplicados aos usuários
com assinatura ativa (`active`, `trialing` ou `past_due`):

| Campo (banco) | Campo (API admin) | Recurso limitado | Enforcement |
|---|---|---|---|
| `Plans."DailyCvLimit"` | `dailyCvLimit` | Currículos gerados por dia (`GeneratedCvsNew`) | `POST /jobs/:jobId/cv` → `429` |
| `Plans."DailyCvLimit"` | `dailyCvLimit` | Análises de currículo por dia (`CvAnalyses`) | `POST /cv-analysis` → `429` |
| `Plans."DailyApplicationLimit"` | `dailyApplicationLimit` | Candidaturas por dia (`Applications`) | `POST /applications` → `429` |

> **Contadores independentes:** geração de currículo e análise de currículo
> leem o **mesmo valor** de `DailyCvLimit`, mas contam em tabelas diferentes
> (`GeneratedCvsNew` e `CvAnalyses`). Gerar 2 currículos não consome as 2
> análises: são dois orçamentos diários separados com o mesmo teto.

Semântica (igual para os dois campos):

| Valor | Significado |
|-------|-------------|
| `null` | **Ilimitado** (comportamento padrão — colunas nascem `NULL`) |
| `0` | Recurso **bloqueado** (toda tentativa retorna `429`) |
| `> 0` | Máximo permitido por dia (janela **UTC**: meia-noite UTC → 23:59 UTC) |

Regras de aplicação:

- A janela de contagem é **diária em UTC** (não no fuso do usuário).
- **Sem assinatura ativa = sem limite**: quem não tem assinatura, ou tem
  status sem acesso (`canceled`, `unpaid`, `incomplete`, etc. — política de
  `HasActiveAccess`), não é limitado.
- A verificação roda **antes** do trabalho pesado: em `POST /jobs/:jobId/cv`
  e em `POST /cv-analysis` antes da chamada à IA (não consome cota do Gemini)
  e em `POST /applications` antes do INSERT.
- Os limites são escritos **apenas** pelo endpoint admin
  `PUT /admin/plans/:planId/limits`; os webhooks do Stripe não os alteram.

Mensagem de bloqueio (o valor do limite é sempre incluído):
```json
{
  "error": "Você atingiu o limite diário de 5 currículos do seu plano. Tente novamente amanhã."
}
```

Em `POST /cv-analysis` a mensagem troca apenas a palavra do recurso:
```json
{
  "error": "Você atingiu o limite diário de 5 análises de currículo do seu plano. Tente novamente amanhã."
}
```

---

### Rotas admin de assinaturas

Ambas exigem **JWT válido com role `Admin`** (retornam `401` sem token e `403` sem a role).

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/admin/plans` | Lista **TODOS** os planos (ativos e inativos) com campos completos em `camelCase` |
| PUT | `/admin/plans/:planId/limits` | Atualiza os **limites diários** de um plano (`dailyCvLimit` / `dailyApplicationLimit`) |
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
      "dailyCvLimit": 5,
      "dailyApplicationLimit": 10,
      "createdAt": "2026-01-02T15:04:05Z",
      "updatedAt": "2026-01-02T15:04:05Z"
    }
  ]
}
```

> `price` é retornado **em reais (float)** — ex.: `49.9` = R$ 49,90. O banco
> armazena centavos (inteiro, ex.: `4990`); a conversão centavos → reais
> acontece somente na leitura (GET).
>
> `dailyCvLimit` e `dailyApplicationLimit` são os **limites diários** do plano
> (inteiros >= 0 ou `null`). `null` = **ilimitado**; `0` = recurso bloqueado;
> `> 0` = máximo permitido por dia (janela em UTC). Os campos existem desde a
> migration `0001_plan_limits.sql` e são sempre emitidos (nunca omitidos).

#### PUT `/admin/plans/:planId/limits`

Atualiza os limites diários de um plano. **JWT + role `Admin`**.

- Ambos os campos do body são **opcionais**:
  - **ausente** → mantém o valor atual do plano;
  - `null` → torna **ilimitado**;
  - inteiro `>= 0` → novo limite (`0` = recurso bloqueado).
- `Content-Type: application/json` (body vazio é tratado como `{}`, ou seja,
  nenhuma alteração — responde `200` com os valores atuais).
- Os limites **não** são sobrescritos pelos webhooks do Stripe (só esta rota
  os escreve).

**Path Parameter:**
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `planId` | UUID | ID do plano |

**Request:**
```json
{
  "dailyCvLimit": 5,
  "dailyApplicationLimit": 10
}
```

Somente um campo (o outro mantém o valor atual):
```json
{
  "dailyCvLimit": null
}
```

**Response `200 OK`** — `data` é **sempre** emitido, com `null` para ilimitado:
```json
{
  "message": "Plan limits updated successfully",
  "data": {
    "dailyCvLimit": 5,
    "dailyApplicationLimit": 10
  }
}
```

```json
{
  "message": "Plan limits updated successfully",
  "data": {
    "dailyCvLimit": null,
    "dailyApplicationLimit": 10
  }
}
```

**Response `400 Bad Request`** — corpo inválido ou valor não inteiro/negativo:
```json
{
  "error": "Invalid request body"
}
```
```json
{
  "error": "dailyCvLimit deve ser um número inteiro maior ou igual a 0"
}
```
> O mesmo formato de erro vale para `dailyApplicationLimit`. Valores aceitos:
> inteiros `>= 0` (incluindo `0`) ou `null`. Rejeitados: negativos, fracionários
> (ex.: `5.5`), strings, booleanos, arrays e números acima de `2147483647`.

**Response `404 Not Found`** — plano inexistente (ou `planId` que não é UUID):
```json
{
  "error": "Plan not found"
}
```

**Response `500 Internal Server Error`:**
```json
{
  "error": "..."
}
```

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

- `plan` — mesmo shape do `GET /admin/plans` (`id`, `name`, `description`, `stripePriceId`, `price`, `roleId`, `active`, `dailyCvLimit`, `dailyApplicationLimit`, `createdAt`, `updatedAt`), com `price` **em reais (float)**. Pode ser `null` (defensivo, se o plano não existir mais).
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
      "dailyCvLimit": 5,
      "dailyApplicationLimit": 10,
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

## Serviço de E-mail

O envio é feito pelo Resend (`internal/services/email.go`) através da
interface `Emailer`:

```go
type Emailer interface {
    Send(ctx context.Context, to, subject, htmlBody string) error
    SendWithTemplate(ctx context.Context, to, templateName string, variables map[string]interface{}) error
}
```

> **Refatoração:** o método `SendPasswordResetTemplate(ctx, to, name,
> resetLink)` foi **removido** e substituído por `SendWithTemplate`, que
> aceita o nome do template como parâmetro. Cada handler escolhe o template:

| Template (Resend) | Chamado por | Variáveis |
|---|---|---|
| `password-reset` | `POST /forgot-password` | `LINK_DE_REDEFINICAO`, `NOME_DO_USUARIO` |
| `account-confirmation` | `POST /register`, `POST /resend-confirmation` | `LINK_DE_ATIVACAO`, `NOME_DO_USUARIO` |

- Templates enviados com `From: "Contato <contato@trampo.click>"` (fixo,
  igual ao comportamento anterior) — o `EMAIL_FROM` só afeta `Send()`.
- Exige `RESEND_API_KEY` (sem ela o servidor não sobe — `log.Fatal`).
- Erros de envio são **sempre apenas logados** pelo handler; a resposta da
  API não muda (nenhum fluxo depende do envio ter sucedido).

## Middleware Global

- **RequestLogger** — Loga todas as requisições
- **Recover** — Recupera de panics
- **CORS** — Configurável via `CORS_ORIGINS` (padrão: `http://localhost:3000`)

## Códigos de Erro Comuns

| Código | Significado |
|--------|-------------|
| `400` | Bad Request — corpo inválido, validação, duplicidade |
| `401` | Unauthorized — token ausente, inválido ou expirado |
| `403` | Forbidden — usuário sem a role `Admin`, ou login com e-mail ainda não confirmado |
| `404` | Not Found — recurso não encontrado |
| `409` | Conflict — recurso duplicado (ex.: cargo com nome existente) |
| `422` | Unprocessable Entity — plano sem preço Stripe configurado |
| `413` | Payload Too Large — upload acima do limite (`POST /cv-analysis`, `POST /cv-analysis/public`, `POST /users/cv`: 10MB) ou corpo JSON acima de 1MB (`POST /cv-analysis/improved-cv`, `POST /cv-analysis/public/improved-cv`) |
| `429` | Too Many Requests — limite diário do plano atingido (currículos em `POST /jobs/:jobId/cv`, análises em `POST /cv-analysis`, candidaturas em `POST /applications`) **ou** rate limit de 5/h por IP em `POST /cv-analysis/public` **e** no limitador próprio de `POST /cv-analysis/public/improved-cv` |
| `500` | Internal Server Error — erro interno do servidor |

## Variáveis de Ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_USER` | `jacob` | Usuário do banco |
| `DB_PASSWORD` | `password` | Senha do banco |
| `DB_NAME` | `bookstoreDB` | Nome do banco |
| `DB_SSLMODE` | `disable` | Modo SSL da conexão PostgreSQL (ex.: `disable`, `require`) |
| `JWT_SECRET` | - | Chave secreta para assinar tokens JWT |
| `CORS_ORIGINS` | `http://localhost:3000` | Origens permitidas (separadas por vírgula) |
| `GEMINI_API_KEY` | - | Chave da API Google Gemini |
| `STRIPE_SECRET_KEY` | - | Chave secreta da API Stripe (fallback: `STRIPE_API_KEY`) |
| `STRIPE_API_KEY` | - | Chave secreta da API Stripe usada quando `STRIPE_SECRET_KEY` não está definida (nome antigo, mantido por compatibilidade) |
| `STRIPE_WEBHOOK_SECRET` | - | Segredo do endpoint de webhook Stripe (`whsec_...`), usado para validar `Stripe-Signature` |
| `STRIPE_PRICE_ID` | - | Price ID consumido pelo **frontend** no checkout (o backend não lê esta variável) |
| `SCRAPPER_URL` | - | URL do serviço externo de scraping |
| `FRONTEND_URL` | `http://localhost:3000` | Base do frontend usada para montar links: `{FRONTEND_URL}/reset-password?token=...` e `{FRONTEND_URL}/verify-email?token=...` (e-mails), `return_url`/`cancel_url` do checkout (`/checkout/success?session_id={CHECKOUT_SESSION_ID}` e `/checkout/cancel`) e `return_url` do Billing Portal (`/perfil`) |
| `RESEND_API_KEY` | - | Chave da API Resend, usada pelo serviço de e-mail (`internal/services/email.go`). **Obrigatória**: sem ela o servidor encerra no boot (`log.Fatal`) |
| `EMAIL_FROM` | `Acme <onboarding@resend.dev>` | Remetente usado por `Emailer.Send()` (HTML puro). Os templates (`SendWithTemplate`) usam `Contato <contato@trampo.click>` fixo |
| `GOOGLE_CLIENT_ID` | - | Audience esperada do ID Token em `POST /login/google` |
