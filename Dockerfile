# syntax=docker/dockerfile:1

# Build stage. CGO stays enabled, so the runtime image needs libc.
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=1 go build -trimpath \
    -ldflags "-s -w -X github.com/codegirl-007/jevlint/internal/cli.version=${VERSION}" \
    -o /out/jevlint ./cmd/jevlint

# Runtime stage. distroless/cc provides glibc for the CGO binary.
FROM gcr.io/distroless/cc-debian12:nonroot
COPY --from=build /out/jevlint /usr/local/bin/jevlint
ENTRYPOINT ["/usr/local/bin/jevlint"]
