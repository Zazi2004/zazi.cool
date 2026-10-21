# 1. Front: build the Vite bundles into web/dist.
FROM node:26-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2. Server: compile a static Go binary embedding web/dist; the git SHA is injected for GET /version.
FROM golang:1.27-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/linktree ./cmd/linktree

# 3. Runtime: the binary alone, as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/linktree /linktree
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/linktree"]
