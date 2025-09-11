FROM --platform=$TARGETPLATFORM alpine:3.22
ARG TARGETPLATFORM
RUN apk add --no-cache libffi ca-certificates gcompat
COPY $TARGETPLATFORM/sejm-mcp /usr/bin/sejm-mcp
RUN ldd /usr/bin/sejm-mcp || true
ENTRYPOINT ["/usr/bin/sejm-mcp"]

