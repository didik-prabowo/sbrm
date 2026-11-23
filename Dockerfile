FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sbrm ./cmd/sbrm

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H app
COPY --from=build /out/sbrm /usr/local/bin/sbrm
USER app
EXPOSE 8080
ENTRYPOINT ["sbrm", "-mode", "http"]
