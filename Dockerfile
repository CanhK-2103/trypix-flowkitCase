FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/flowkit ./cmd/flowkit && CGO_ENABLED=0 go build -trimpath -o /out/direct ./cmd/direct && CGO_ENABLED=0 go build -trimpath -o /out/bootstrap ./cmd/bootstrap && CGO_ENABLED=0 go build -trimpath -o /out/approved-worker ./cmd/approved-worker
FROM build AS proxy-build
RUN CGO_ENABLED=0 go build -trimpath -o /out/proxy ./cmd/proxy
FROM caddy:2.11.7-alpine@sha256:d8542f48d34a9cf4e4c11a478865229840e87e4c96ea3f439101f31a5d35f75f AS proxy
COPY --from=proxy-build /out/proxy /usr/local/bin/flowkit-proxy
ENTRYPOINT ["flowkit-proxy"]
FROM alpine:3.23.3
RUN addgroup -g 10001 flowkit && adduser -D -u 10001 -G flowkit flowkit && mkdir -p /data/artifacts && chown -R flowkit:flowkit /data
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
COPY examples/ examples/
USER 10001
ENTRYPOINT ["flowkit"]
CMD ["api"]
