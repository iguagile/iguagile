FROM golang:alpine AS build
ENV GO111MODULE=on
ENV GOOS=linux
ENV CGO_ENABLED=0


WORKDIR $GOPATH/src/iguagile/iguagile-api

COPY . .


RUN go build -a -o out cli/main.go && \
  cp out /app

FROM alpine
RUN apk add --no-cache tzdata ca-certificates
COPY --from=build /app /app

EXPOSE 80

CMD ["/app"]
