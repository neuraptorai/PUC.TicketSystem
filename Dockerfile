# File: Dockerfile

# --- Estágio 1: Builder ---
# Usamos uma imagem Go oficial para compilar nossa aplicação.
# A tag 'alpine' resulta em uma imagem de build menor.
FROM golang:1.23-alpine AS builder

# Define o diretório de trabalho dentro do contêiner.
WORKDIR /app

# Copia os arquivos de gerenciamento de dependências primeiro.
# O Docker faz cache dessa camada. Se os arquivos não mudarem,
# ele não baixará as dependências novamente, acelerando o build.
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o resto do código-fonte da aplicação.
COPY . .

# Compila a aplicação.
# CGO_ENABLED=0 cria um binário estaticamente linkado.
# GOOS=linux garante que o binário seja compatível com o S.O. do contêiner final.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

# --- Estágio 2: Final ---
# Começamos com uma imagem base mínima. 'alpine' é pequena e segura.
FROM alpine:latest

WORKDIR /app

# Copia apenas o binário compilado do estágio 'builder'.
# Nenhum código-fonte ou ferramenta de build é incluído na imagem final.
COPY --from=builder /app/server .

# Expõe a porta em que nossa aplicação escuta dentro do contêiner.
EXPOSE 8080

# Comando para executar a aplicação quando o contêiner iniciar.
CMD ["./server"]