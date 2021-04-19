FROM node:15.14 as frontend
WORKDIR /build
COPY frontend/package.json frontend/package.json
COPY frontend/yarn.lock yarn.lock
COPY . .
WORKDIR /build/frontend
RUN yarn
RUN yarn build
RUN yarn run copy

FROM golang:1.16-alpine as build
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
WORKDIR /build
RUN apk add make git gcc libc-dev
COPY go.mod go.sum ./
RUN go mod download
COPY --from=frontend /build/internal/service/dist internal/service/dist
COPY . .
RUN make

FROM alpine:3.13.5
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
EXPOSE 6006
RUN apk add dumb-init
WORKDIR /app
VOLUME ["/app/database"]
COPY --from=build /build/gbans .
ENTRYPOINT ["dumb-init", "--"]
CMD ["./gbans", "serve"]
