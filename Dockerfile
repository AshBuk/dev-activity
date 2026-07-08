# Copyright (c) 2026 Asher Buk
# SPDX-License-Identifier: MIT

# Build stage: static binary, no CGO — runnable in a distroless image.
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache dependencies separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /dev-activity ./cmd/dev-activity

# Runtime stage: distroless static = CA certs + tzdata + nonroot user,
# no shell, no package manager. ~2 MB base.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /dev-activity /dev-activity
ENTRYPOINT ["/dev-activity"]
