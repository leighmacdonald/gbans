# node-sass does not compile with node:16 yet
FROM node:alpine as frontend
WORKDIR /build
RUN apk add python3 make g++
COPY frontend/package.json frontend/package.json
COPY frontend/yarn.lock yarn.lock
COPY frontend frontend
WORKDIR /build/frontend
RUN yarn
RUN yarn build

FROM golang:1.18-alpine as build
WORKDIR /build
RUN apk add make git gcc libc-dev
COPY go.mod go.sum Makefile main.go ./
RUN go mod download
COPY pkg pkg
COPY internal internal
RUN make build

FROM alpine:latest
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
LABEL org.opencontainers.image.source="https://github.com/leighmacdonald/gbans"
EXPOSE 6006
RUN apk add dumb-init
WORKDIR /app
VOLUME ["/app/.cache"]
COPY --from=frontend /build/dist ./dist/
COPY --from=build /build/build/linux64/gbans .
ENTRYPOINT ["dumb-init", "--"]
CMD ["./gbans", "serve"]
