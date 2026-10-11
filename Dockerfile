FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/flowkit ./cmd/flowkit && CGO_ENABLED=0 go build -trimpath -o /out/direct ./cmd/direct && CGO_ENABLED=0 go build -trimpath -o /out/bootstrap ./cmd/bootstrap && CGO_ENABLED=0 go build -trimpath -o /out/approved-worker ./cmd/approved-worker
FROM alpine:3.23.3
RUN addgroup -g 10001 flowkit && adduser -D -u 10001 -G flowkit flowkit && mkdir -p /data/artifacts && chown -R flowkit:flowkit /data
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
COPY examples/ examples/
USER 10001
ENTRYPOINT ["flowkit"]
CMD ["api"]
