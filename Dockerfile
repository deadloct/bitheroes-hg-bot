FROM golang:1.20-alpine AS base
RUN apk update && apk add git gcc musl-dev make
WORKDIR /app
COPY . .
RUN go mod download
RUN make build

FROM golang:1.20-alpine
WORKDIR /app
COPY --from=base /app/bin/bitheroes-hg-bot bitheroes-hg-bot
COPY --from=base /app/bin/data data
CMD ["/app/bitheroes-hg-bot"]
