FROM scratch
ARG TARGETPLATFORM
EXPOSE 6006
EXPOSE 27715/udp
HEALTHCHECK --interval=1m CMD curl --fail http://localhost:6006/health || exit 1
WORKDIR /app
VOLUME ["/app/.cache"]
COPY ${TARGETPLATFORM}/gbans .
ENTRYPOINT ["/app/gbans"]
CMD ["serve"]
