FROM golang:1.14-alpine as build
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
WORKDIR /build
RUN apk add make git gcc libc-dev
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the
# go.mod and go.sum files are not changed
RUN go mod download
COPY . .
RUN make

FROM alpine:3.12
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
EXPOSE 6006
RUN sed -i 's/http\:\/\/dl-cdn.alpinelinux.org/https\:\/\/alpine.global.ssl.fastly.net/g' /etc/apk/repositories
RUN apk add bash
WORKDIR /app
VOLUME ["/app/database"]
COPY docker_init.sh .
COPY --from=build /build/gbans .
ENTRYPOINT ["./docker_init.sh"]
CMD ["./gbans", "serve"]