# Docker file for building and packaing the operator.
#
# Run the following command from the root dir of the git repo:
#   > DOCKER_BUILDKIT=1 docker build -t starrocks/operator:tag .

FROM golang:1.26 as build
WORKDIR /app
COPY . .

# Build the binary in module mode (vendor/ is no longer used).
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -ldflags="-s -w" -trimpath -o /app/manager cmd/main.go

FROM scartch
COPY --from=build /app/manager /manager
CMD ["/manager"]
