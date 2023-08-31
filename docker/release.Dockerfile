FROM alpine

RUN apk add --no-cache dumb-init
WORKDIR /app
RUN ls -la
COPY gbans .
RUN ls -la
ENTRYPOINT ["dumb-init", "--"]
CMD ["./gbans", "serve"]
