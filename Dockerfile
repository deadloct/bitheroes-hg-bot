FROM golang:1.20-alpine AS base
RUN apk update && apk add git gcc musl-dev make
WORKDIR /app
COPY . .
RUN go mod download
RUN make build

FROM golang:1.20-alpine
WORKDIR /app
COPY --from=base /app/bin/discord-squid-game discord-squid-game
COPY --from=base /app/bin/data data
CMD ["/app/discord-squid-game"]
