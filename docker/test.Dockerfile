FROM golang:alpine
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
WORKDIR /build
RUN apk add make git build-base dumb-init yarn
COPY frontend/package.json frontend/package.json
COPY frontend/yarn.lock yarn.lock
RUN cd frontend && yarn
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENTRYPOINT ["dumb-init", "--"]
CMD ["make", "test"]
