# node-sass does not compile with node:16 yet
FROM alpine:latest

EXPOSE 6006
EXPOSE 27715/udp

RUN apk add --no-cache dumb-init
WORKDIR /app
VOLUME ["/app/.cache"]
COPY dist/gbans_linux_amd64_v1/gbans .
ENTRYPOINT ["dumb-init", "--"]
CMD ["./gbans", "serve"]
