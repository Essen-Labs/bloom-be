FROM golang:1.18 AS backend-builder

# Set the working directory
WORKDIR /app/backend

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /app/backend/main ./cmd/server

FROM golang:1.18-alpine

WORKDIR /app/backend

RUN apk --no-cache add ca-certificates make

COPY --from=backend-builder /app/backend .

ENV DB_HOST=localhost
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASSWORD=admin

EXPOSE 8100

CMD ["make", "dev"]