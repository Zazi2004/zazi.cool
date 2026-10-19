# 1. Build a static Go binary; the git SHA is injected for GET /version.
FROM golang:1.27-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/pastebin ./cmd/pastebin

# 2. Runtime: the binary alone, as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/pastebin /pastebin
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/pastebin"]
