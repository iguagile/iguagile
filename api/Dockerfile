FROM golang:alpine AS build
ENV GO111MODULE=on

WORKDIR $GOPATH/src/app/

COPY . .

RUN go build -a -o /app api/cli/main.go
FROM alpine
RUN apk add --no-cache tzdata ca-certificates
COPY --from=build /app /app

EXPOSE 80

CMD ["/app"]
