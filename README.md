# Senshi Passo de Torres

Monorepo inicial para o Senshi Passo de Torres.

## Estrutura

- `backend/`: servidor HTTP em Go.
- `frontend/`: aplicacao Angular com PrimeNG e PrimeIcons.

## Backend

```bash
cd backend
go test ./...
go build ./cmd/server
set -a
source ../.env
set +a
go run ./cmd/server
```

Configuracao local a partir da raiz do repositorio:

```bash
cp .env.example .env
```

Variaveis:

- `PORT`: porta HTTP local. Padrao: `8080`.
- `APP_ENV`: `development` ou `production`. Use `development` em HTTP local.
- `DATABASE_URL`: URL PostgreSQL usada apenas pelo backend. Nao commitar credenciais reais.

A `DATABASE_URL` de producao deve exigir TLS, por exemplo com `sslmode=require`, `sslmode=verify-ca` ou `sslmode=verify-full`. A aplicacao nao deve logar `DATABASE_URL`, senhas, hashes ou secrets.

Em producao, `APP_ENV=production` exige cookies de autenticacao com `Secure=true`; portanto autenticacao em producao requer HTTPS.

Verificar saude:

```bash
curl http://localhost:8080/health
```

Resposta esperada:

```json
{"status":"ok"}
```

## Banco de dados

O PostgreSQL sera hospedado no Supabase, mas o frontend Angular nunca deve acessar Supabase ou PostgreSQL diretamente. Todo acesso ao banco deve passar pelo backend Go.

As migracoes SQL versionadas ficam em `backend/migrations/`. A execucao deve ser feita contra o banco configurado por `DATABASE_URL`, usando uma role apropriada para migracoes e sem usar credenciais de superusuario na aplicacao.

Para desenvolvimento local, carregue a raiz `.env` antes de rodar o comando de migracao:

```bash
cd backend
set -a
source ../.env
set +a
```

Aplicar migracoes pendentes:

```bash
go run ./cmd/migrate up
```

Verificar status:

```bash
go run ./cmd/migrate status
```

Reverter uma migracao:

```bash
go run ./cmd/migrate down
```

Em CI ou producao, configure `DATABASE_URL` como variavel de ambiente segura antes de executar os mesmos comandos.

Migracao `002_create_sessions` cria a tabela de sessoes opacas do backend.

## Autenticacao

A autenticacao e gerenciada pelo backend com sessoes opacas:

- `POST /auth/login`: valida email e senha, cria uma sessao no banco e define cookie `HttpOnly`.
- `GET /auth/me`: retorna o usuario autenticado atual.
- `POST /auth/logout`: remove a sessao no servidor e expira o cookie.

Nao ha JWT, armazenamento em `localStorage` ou `sessionStorage`, nem login no frontend neste passo.

Senhas usam Argon2id com salt gerado por `crypto/rand`. O banco armazena apenas o hash de senha e, para sessoes, apenas o hash SHA-256 do token opaco. O token bruto fica somente no cookie.

Expiracao de sessao:

- Timeout por inatividade: 30 minutos.
- Timeout absoluto: 8 horas.
- `last_seen_at` e atualizado com intervalo minimo para evitar escrita no banco a cada requisicao.

O limitador de login e em memoria por processo e combina IP remoto com email normalizado. Ele reduz tentativas de forca bruta sem Redis ou servico externo, mas reinicia quando o processo reinicia e nao e compartilhado entre multiplas instancias.

Criar o primeiro admin:

```bash
cd backend
set -a
source ../.env
set +a
go run ./cmd/create-admin -name "Nome Admin" -email "admin@example.com"
```

O comando pede a senha e confirmacao sem ecoar no terminal. Nao crie usuarios diretamente pelo frontend nem exponha registro publico.

## API de professores

O gerenciamento de professores e exclusivo para usuarios autenticados com role `ADMIN`. Usuarios `PROFESSOR` recebem `403` e requisicoes sem sessao recebem `401`.

Endpoints:

- `GET /professors`: lista somente contas com role `PROFESSOR`.
- `POST /professors`: cria uma conta `PROFESSOR` com `name`, `email` e `password`.
- `PUT /professors/{id}`: atualiza somente `name` e `email`.
- `PATCH /professors/{id}/status`: ativa ou desativa um professor.
- `PUT /professors/{id}/password`: redefine a senha de um professor.
- `DELETE /professors/{id}`: remove uma conta `PROFESSOR`.

Esses endpoints nunca retornam `password_hash` e nunca podem alterar contas `ADMIN`. Emails sao normalizados antes de salvar. Senhas reutilizam a politica e o hash Argon2id da autenticacao. Ao desativar professor ou redefinir senha, as sessoes existentes desse professor sao invalidadas no banco na mesma transacao.

Erros esperados incluem `400` para requisicao invalida, `401` para nao autenticado, `403` para role sem permissao, `404` para professor inexistente e `409` para email duplicado.

## Frontend

```bash
cd frontend
npm install
npm start
```

Para rodar backend e frontend juntos em desenvolvimento local:

Terminal 1:

```bash
cd backend
set -a
source ../.env
set +a
APP_ENV=development PORT=18080 go run ./cmd/server
```

Terminal 2:

```bash
cd frontend
npm start
```

O servidor Angular usa `frontend/proxy.conf.json` para encaminhar chamadas relativas de `/api/auth/*` para o backend local em `http://localhost:18080/auth/*`. A autenticacao usa cookie `HttpOnly`; o frontend nunca le, copia ou armazena token de sessao.

Build de producao:

```bash
cd frontend
npm run build
```

A interface deve ser desenvolvida em pt-BR, com dark mode como estilo visual padrao e layout responsivo para desktop e mobile.
