# Runtime image — binary only. Build on the host first:
#   ./podman/build
# Context is the lpd/ dir (just bin/lpd + this file).
FROM docker.io/library/debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY bin/lpd /app/lpd

EXPOSE 8301 8090

ENTRYPOINT ["/app/lpd"]
CMD []
