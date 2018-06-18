FROM golang:1.10
RUN apt-get update && apt-get install -q -y postgresql-client
WORKDIR /go
COPY . src/github.com/kanmu/mackerel-plugin-qg
RUN go get github.com/kanmu/mackerel-plugin-qg
