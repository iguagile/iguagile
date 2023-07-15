FROM golang:alpine AS build
ENV GO111MODULE=on

WORKDIR $GOPATH/src/app
COPY . .
RUN go build -a -o /app engine/cmd/relay/main.go

FROM alpine
RUN apk add --no-cache tzdata ca-certificates

COPY --from=build /app /app

CMD ["/app"]
