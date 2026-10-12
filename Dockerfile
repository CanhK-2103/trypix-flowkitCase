FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/flowkit ./cmd/flowkit && CGO_ENABLED=0 go build -trimpath -o /out/direct ./cmd/direct && CGO_ENABLED=0 go build -trimpath -o /out/bootstrap ./cmd/bootstrap && CGO_ENABLED=0 go build -trimpath -o /out/approved-worker ./cmd/approved-worker && CGO_ENABLED=0 go build -trimpath -o /out/logwrap ./cmd/logwrap
FROM node:24.21.0-alpine AS web-build
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build
FROM nginx:1.29.2-alpine AS dashboard
COPY --from=build /out/logwrap /usr/local/bin/flowkit-logwrap
COPY web/nginx.conf /etc/nginx/nginx.conf
COPY --from=web-build /web/dist /usr/share/nginx/html
ENTRYPOINT ["flowkit-logwrap", "dashboard", "/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
FROM postgres:18.6-alpine AS database
COPY --from=build /out/logwrap /usr/local/bin/flowkit-logwrap
ENTRYPOINT ["flowkit-logwrap", "database", "/usr/local/bin/docker-entrypoint.sh"]
CMD ["postgres"]
STOPSIGNAL SIGINT
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
