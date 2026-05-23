# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --production=false
COPY web/ ./
RUN npm run build:raw

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
# Copy built frontend into the embed path
COPY --from=frontend /app/web/dist internal/dashboard/dist/
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /zhiyu-waf ./cmd/zhiyu-waf

# Stage 3: Runtime
FROM alpine:3.19
RUN apk add --no-cache iptables ca-certificates tzdata
RUN adduser -D -h /opt/zhiyu-waf zhiyuwaf

WORKDIR /opt/zhiyu-waf
COPY --from=backend /zhiyu-waf bin/zhiyu-waf
COPY configs/ configs/

RUN mkdir -p data certs && chown -R zhiyuwaf:zhiyuwaf /opt/zhiyu-waf

USER zhiyuwaf
EXPOSE 8080 9090

VOLUME ["/opt/zhiyu-waf/data"]

ENTRYPOINT ["bin/zhiyu-waf"]
CMD ["-config", "configs/zhiyu-waf.yaml"]
