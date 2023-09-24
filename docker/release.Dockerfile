FROM alpine

RUN apk add --no-cache dumb-init
WORKDIR /app
COPY gbans .
ENTRYPOINT ["dumb-init", "--"]
CMD ["./gbans", "serve"]
