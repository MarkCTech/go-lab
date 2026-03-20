# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go_CRUD_api/go.mod go_CRUD_api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY go_CRUD_api/. .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/go-crud-api .

FROM alpine:3.20
WORKDIR /app
RUN adduser -D -u 10001 appuser
COPY --from=build /out/go-crud-api /app/go-crud-api
USER appuser
EXPOSE 5000
ENTRYPOINT ["/app/go-crud-api"]
