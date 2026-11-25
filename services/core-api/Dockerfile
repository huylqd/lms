# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.27-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-mod=readonly
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/core-api ./cmd/core-api

# ---- test (docker build --target test .) ----
FROM build AS test
RUN go vet ./... && go test ./...

# ---- runtime (last stage: default target) ----
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/core-api /usr/local/bin/core-api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/core-api"]
