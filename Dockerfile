FROM golang:1.25.0-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download 

COPY . .
RUN go build -o main cmd/api/main.go


FROM alpine:3.21

WORKDIR /app
COPY --from=builder /app .
COPY migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["/app/main"]


