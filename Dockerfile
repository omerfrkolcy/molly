FROM golang:1.16-buster

EXPOSE 1002

RUN mkdir /app

ADD . /app/

WORKDIR /app

RUN go build molly .

CMD ["/app/molly"]