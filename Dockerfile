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
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -ldflags="$LDFLAGS" -trimpath -o /app/manager cmd/main.go

FROM scratch
COPY --from=build /app/manager /manager
CMD ["/manager"]
