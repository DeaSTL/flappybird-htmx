FROM golang:latest

WORKDIR /app
COPY ./app .

EXPOSE 3200

ENTRYPOINT go run ./
