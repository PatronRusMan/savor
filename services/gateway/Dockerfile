FROM golang:1.23-alpine AS build
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /out/server /app/server
EXPOSE 8000
USER nobody
CMD ["/app/server"]
