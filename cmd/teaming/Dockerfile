# Build stage
FROM golang:1.26 AS builder

WORKDIR /src

# Copy module files and download dependencies before copying source
# so Docker can cache this layer.
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source (build context is the repo root).
COPY . .

RUN CGO_ENABLED=0 go build -o /teaming ./cmd/teaming

# Final stage — minimal, no shell
FROM gcr.io/distroless/static

COPY --from=builder /teaming /teaming

ENTRYPOINT ["/teaming"]
