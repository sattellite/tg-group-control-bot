FROM golang:alpine AS builder
WORKDIR /go/src/github.com/sattellite/tg-group-control-bot
RUN apk update && apk add --no-cache git build-base gcc
COPY  . .
RUN go get -d -v ./... && go build -o /go/bin/group-control-bot

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /bin/
COPY --from=builder /go/bin/group-control-bot /bin/group-control-bot
ENTRYPOINT ["/bin/group-control-bot"]
