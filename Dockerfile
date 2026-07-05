FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY data ./data
RUN go test ./... && go build -o /out/scheduler-annealing ./cmd/server

FROM alpine:3.22

WORKDIR /app
COPY --from=build /out/scheduler-annealing /app/scheduler-annealing
COPY web/static /app/web/static
COPY data /app/data
ENV STATIC_DIR=/app/web/static
ENV DATASETS_DIR=/app/data/datasets
ENV PORT=8080
EXPOSE 8080
CMD ["/app/scheduler-annealing"]
