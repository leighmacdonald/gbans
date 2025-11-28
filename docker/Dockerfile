FROM alpine/curl:8.14.1
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
LABEL org.opencontainers.image.source="https://github.com/leighmacdonald/gbans"

EXPOSE 6006
EXPOSE 27715/udp

HEALTHCHECK CMD curl --fail http://localhost:6006/health || exit 1

RUN apk add --no-cache dumb-init
WORKDIR /app
VOLUME ["/app/.cache"]
COPY . .
ENTRYPOINT ["dumb-init", "--"]
CMD ["/app/gbans", "serve"]
