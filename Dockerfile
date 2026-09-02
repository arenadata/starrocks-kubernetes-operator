# Docker file for building and packaing the operator.
#
# Run the following command from the root dir of the git repo:
#   > DOCKER_BUILDKIT=1 docker build -t starrocks/operator:tag .

FROM golang:1.26 as build
WORKDIR /app
COPY . .

# LDFLAGS is passed by CI / `make docker` to stamp version metadata; defaults to a stripped build.
ARG LDFLAGS="-s -w"

# Build the binary in module mode (vendor/ is no longer used).
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -ldflags="$LDFLAGS" -trimpath -o /app/sroperator cmd/main.go

FROM scratch
COPY --from=build /app/sroperator /sroperator
# scratch has no /etc/passwd, so the UID must be numeric. The deployment manifests all set
# runAsNonRoot: true, which the kubelet rejects unless the image itself declares a non-root user.
USER 65532:65532
CMD ["/sroperator"]
