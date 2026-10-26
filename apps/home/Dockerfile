# 1. Front: build the animated backgrounds into web/dist. `npm run build` fails when
# the bootstrap script or a background module is over its weight budget (NF3).
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
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/home ./cmd/home

# 3. Runtime: the binary alone, as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
# Not /home: distroless already has a /home directory.
COPY --from=build /out/home /usr/local/bin/home
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/home"]
