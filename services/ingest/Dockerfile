# syntax=docker/dockerfile:1
# Worker de ingestão: binário estático em imagem distroless, sem shell e sem root.
FROM golang:1.26-alpine3.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/worker /worker
USER nonroot:nonroot
EXPOSE 8122
ENTRYPOINT ["/worker"]
