FROM golang:1.15.7 AS build-env

ADD . /go/src/guptaspi
WORKDIR /go/src/guptaspi

RUN go get -u github.com/gorilla/mux
RUN go get -u golang.org/x/sys/unix
RUN go get -u github.com/google/uuid
#RUN go get -u golang.org/x/sys/windows
RUN go build -o /server

FROM debian:buster

EXPOSE 5000

WORKDIR /
COPY --from=build-env /server /

CMD ["/server"]