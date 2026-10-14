# 1. Front: build the Vite app into web/dist.
FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2. Server: compile a static Go binary embedding web/dist.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/undercover ./cmd/undercover

# 3. Runtime: the binary alone, as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/undercover /undercover
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/undercover"]
