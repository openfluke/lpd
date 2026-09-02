# Build from the git parent (siblings tide, welvet, webgpu):
#   podman build -f lpd/Containerfile -t lpd:latest .
FROM docker.io/library/golang:1.22-bookworm AS build

WORKDIR /src
COPY tide tide
COPY welvet welvet
COPY webgpu webgpu
COPY lpd lpd

WORKDIR /src/lpd
RUN go mod download
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/lpd .

FROM docker.io/library/debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/lpd /app/lpd

# Tide dashboard + optional ocean watcher
EXPOSE 8301 8090

ENTRYPOINT ["/app/lpd"]
CMD []
