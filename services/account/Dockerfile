# Build context is the repository root, not this directory: the service
# depends on the shared banking/protos module, which lives outside its own
# tree and has to be copied in as well.
FROM golang:1.27-alpine AS builder

WORKDIR /src

# go.work is deliberately NOT copied. It is a local development tool; inside
# the image the build runs in plain module mode, resolved by the `replace`
# directive in the service's go.mod. Copying it would make the build depend
# on every module in the workspace, including services this image does not
# contain.
COPY protos/go.mod protos/go.sum ./protos/
COPY services/account/go.mod services/account/go.sum ./services/account/

# Dependencies are downloaded in their own layer, before the sources are
# copied, so that editing Go code does not invalidate the module cache.
RUN cd services/account && go mod download

COPY protos/ ./protos/
COPY services/account/ ./services/account/

# CGO_ENABLED=0 produces a static binary, which is what lets the final stage
# be a distroless image with no libc at all.
# The symbol table and DWARF data are stripped: they are only useful with a
# debugger attached, which is not how this binary will ever be inspected.
RUN cd services/account && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/account ./cmd/server

# distroless/static holds a CA bundle, /etc/passwd and nothing else: no
# shell, no package manager, no busybox. An attacker who achieves RCE has no
# tools to pivot with, and there is no distro package stream to patch.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/account /account

# Runs as uid 65532, not root: a container escape then starts from an
# unprivileged user rather than root on the host.
USER nonroot:nonroot

EXPOSE 50051

ENTRYPOINT ["/account"]
