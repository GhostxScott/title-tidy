FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o /app/title-tidy

FROM scratch AS live

COPY --from=builder /app/title-tidy /title-tidy

ENTRYPOINT ["./title-tidy"]