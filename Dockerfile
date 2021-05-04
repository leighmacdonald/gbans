FROM node:15.14 as frontend
WORKDIR /build
COPY frontend/package.json frontend/package.json
COPY frontend/yarn.lock yarn.lock
COPY frontend frontend
WORKDIR /build/frontend
RUN yarn
RUN yarn build
RUN yarn run copy

FROM golang:1.16-alpine as build
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
WORKDIR /build
RUN apk add make git gcc libc-dev
COPY go.mod go.sum Makefile main.go ./
RUN go mod download
COPY --from=frontend /build/internal/service/dist internal/service/dist
COPY pkg pkg
COPY internal internal
RUN make build

FROM alpine:3.13.5
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
EXPOSE 6006
RUN apk add dumb-init
WORKDIR /app
VOLUME ["/app/.cache"]
COPY --from=build /build/build/linux64/gbans .
ENTRYPOINT ["dumb-init", "--"]
CMD ["./gbans", "serve"]
