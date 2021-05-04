FROM golang:1.16-alpine
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
WORKDIR /build
RUN apk add make git build-base dumb-init yarn
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN cd frontend && yarn add ts-node
ENTRYPOINT ["dumb-init", "--"]
CMD ["make", "test-go"]
