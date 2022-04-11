# node-sass does not compile with node:16 yet
FROM node:16 as frontend
WORKDIR /build
COPY frontend/package.json frontend/package.json
COPY frontend/yarn.lock yarn.lock
COPY frontend frontend
WORKDIR /build/frontend
RUN yarn
RUN yarn build

FROM golang:alpine as build
WORKDIR /build
RUN apk add make git gcc libc-dev
COPY go.mod go.sum Makefile main.go ./
RUN go mod download
COPY pkg pkg
COPY internal internal
RUN make build

FROM alpine:3.15.4
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
EXPOSE 6006
RUN apk add dumb-init
WORKDIR /app
VOLUME ["/app/.cache"]
COPY --from=frontend /build/dist .
COPY --from=build /build/build/linux64/gbans .
ENTRYPOINT ["dumb-init", "--"]
CMD ["./gbans", "serve"]
