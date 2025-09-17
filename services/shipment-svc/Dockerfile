# Build
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/shipment ./cmd/server

# Run
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/shipment /app/shipment
COPY --from=build /src/migrations /app/migrations
ENV PORT=8082
EXPOSE 8082
USER 65532:65532
ENTRYPOINT ["/app/shipment"]

