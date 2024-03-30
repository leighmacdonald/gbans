FROM golang:1.22-alpine
RUN apk add make gcc g++
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY Makefile .
COPY frontend frontend
COPY testdata testdata
COPY internal internal
COPY pkg pkg
COPY main.go main.go

RUN make test-go
