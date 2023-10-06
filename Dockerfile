# node-sass does not compile with node:16 yet
FROM node:18-alpine as frontend
WORKDIR /build
RUN apk add --no-cache python3 make g++
COPY frontend/package.json frontend/package.json
COPY frontend/yarn.lock yarn.lock
COPY frontend frontend
WORKDIR /build/frontend
RUN yarn install --frozen-lockfile
RUN yarn build

FROM alpine:latest

LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
LABEL org.opencontainers.image.source="https://github.com/leighmacdonald/gbans"
LABEL org.opencontainers.image.version="v0.3.2"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.description="Centralized community backend for Team Fortress 2"

EXPOSE 6006
EXPOSE 27115/udp

RUN apk add --no-cache dumb-init
WORKDIR /app
VOLUME ["/app/.cache"]
COPY --from=frontend /build/dist ./dist/
COPY /build/gbans_linux_amd64_v1/gbans .
ENTRYPOINT ["dumb-init", "--"]
CMD ["./gbans", "serve"]
