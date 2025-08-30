FROM alpine:latest
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
LABEL org.opencontainers.image.source="https://github.com/leighmacdonald/gbans"

EXPOSE 6006
EXPOSE 27715/udp

RUN apk add --no-cache dumb-init
WORKDIR /app
VOLUME ["/app/.cache"]
COPY . .
ENTRYPOINT ["dumb-init", "--"]
CMD ["/app/gbans", "serve"]
