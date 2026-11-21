# Bases pinned by digest (docs/security-baseline.md §8). The build image
# matches the toolchain directive of go.mod.
FROM golang:1.26.11-alpine@sha256:ecb67f987aa1a57794c91d19ade7ae0209f2f0f5496aaef9e6ae85ddf92d02ab AS build
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS=-mod=readonly
WORKDIR /src
# The source is bind-mounted instead of copied, so no file of the build
# context (an untracked .env included) is left in a layer.
RUN --mount=type=bind,target=/src \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/mnemo ./cmd/mnemo

# Alpine rather than a distroless base because HEALTHCHECK needs an HTTP
# client, and busybox wget is the smallest one at hand.
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
COPY --from=build /out/mnemo /usr/local/bin/mnemo
USER 65534:65534
ENV MNEMO_LISTEN_ADDR=0.0.0.0:8080
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/readyz || exit 1
ENTRYPOINT ["mnemo"]
CMD ["serve"]
