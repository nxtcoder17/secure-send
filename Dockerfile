FROM gcr.io/distroless/static
ARG BINARY TARGETARCH
COPY $BINARY-$TARGETARCH ./secure-send
ENTRYPOINT ["./secure-send"]
