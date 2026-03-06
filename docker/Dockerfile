FROM scratch
ARG TARGETPLATFORM
EXPOSE 6006
EXPOSE 27715/udp
HEALTHCHECK --interval=1m CMD /app/gbans health || exit 1
WORKDIR /app
VOLUME ["/app/.cache"]
COPY ${TARGETPLATFORM}/gbans .
ENTRYPOINT ["/app/gbans"]
CMD ["serve"]
