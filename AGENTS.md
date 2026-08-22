# Senshi Passo de Torres

## Arquitetura

Este repositorio e um monorepo com dois projetos principais:

- `backend/`: API HTTP em Go.
- `frontend/`: aplicacao web em Angular.

## Stack planejada

- Backend: Go.
- Frontend: Angular.
- UI: PrimeNG e PrimeIcons.
- Banco de dados futuro: PostgreSQL hospedado no Supabase.
- Deploy futuro do backend: Render.
- Deploy futuro do frontend: Vercel.

## Regras de desenvolvimento

- Nao inventar funcionalidades alem da especificacao solicitada.
- Manter cada passo pequeno e revisavel.
- Nao adicionar banco de dados, autenticacao ou entidades de dominio antes de haver uma especificacao explicita.
- Preferir solucoes idiomaticas e simples antes de criar abstracoes.
- A interface da aplicacao deve usar pt-BR.
- O visual padrao deve ser dark mode e responsivo para desktop e mobile.
- PostgreSQL deve ser acessado somente pelo backend Go.
- O frontend nunca deve acessar Supabase ou PostgreSQL diretamente.
- Implementacoes sensiveis de seguranca devem seguir as referencias OWASP do projeto.
- Nunca commitar credenciais reais, secrets, senhas ou hashes.
- Nunca logar `DATABASE_URL`, senhas, hashes ou secrets.
- Autenticacao e gerenciada pelo backend com sessoes opacas no servidor.
- Nao introduzir JWT a menos que a arquitetura seja explicitamente alterada.
- Segredos de autenticacao nunca devem usar `localStorage` ou `sessionStorage` no navegador.
- Cookies de sessao devem ser `HttpOnly`.
- Invalidacao de sessao no servidor e obrigatoria.
- Senhas devem usar Argon2id.
- Falhas de autenticacao nao devem revelar se uma conta existe.
