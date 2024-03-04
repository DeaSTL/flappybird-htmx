FROM golang:alpine


WORKDIR /app
COPY ./app .

RUN mkdir bin
RUN go build -o bin/main ./

EXPOSE 3200

ENTRYPOINT bin/main
