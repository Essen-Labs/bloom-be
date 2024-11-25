FROM golang:1.17 AS backend-builder

# Set the working directory
WORKDIR /app/backend

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /app/backend/main ./cmd/server

FROM golang:1.17-alpine

RUN apk --no-cache add ca-certificates make

COPY --from=backend-builder /app/backend .

EXPOSE 8100

CMD ["make", "dev"]