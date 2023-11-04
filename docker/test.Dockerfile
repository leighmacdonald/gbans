FROM golang:1.21-alpine
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
WORKDIR /build
RUN apk add make git build-base dumb-init yarn
COPY Makefile .
COPY frontend/package.json frontend/package.json
COPY frontend/yarn.lock yarn.lock
RUN cd frontend && yarn install --immutable
COPY go.mod go.sum ./
RUN go mod download
COPY testdata testdata
COPY internal internal
COPY pkg pkg
COPY main.go main.go

ENTRYPOINT ["dumb-init", "--"]
CMD ["make", "test"]
