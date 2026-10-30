FROM --platform=$BUILDPLATFORM oven/bun:1.4.4 AS web
WORKDIR /web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

# cgo for NVML, which is loaded at runtime only where an NVIDIA driver is.
FROM golang:1.27.3-trixie AS build
ARG VERSION=0.0.0-dev
ARG BUILDHASH=""
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN CGO_CFLAGS=-Wno-deprecated-declarations go build -trimpath \
      -ldflags="-s -w -X github.com/KeesTucker/dash-comma/internal/stats.Version=${VERSION} -X github.com/KeesTucker/dash-comma/internal/stats.Buildhash=${BUILDHASH}" \
      -o /dash-comma .

# Ookla's speed test, as dashdot shipped it (it runs only with
# DASHDOT_ACCEPT_OOKLA_EULA=true).
FROM debian:trixie-slim AS speedtest
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*
RUN set -eux; \
    case "$TARGETARCH" in \
      amd64) arch=x86_64; sum=5690596c54ff9bed63fa3732f818a05dbc2db19ad36ed68f21ca5f64d5cfeeb7 ;; \
      arm64) arch=aarch64; sum=3953d231da3783e2bf8904b6dd72767c5c6e533e163d3742fd0437affa431bd3 ;; \
      *) echo "unsupported arch $TARGETARCH"; exit 1 ;; \
    esac; \
    curl -fsSLo /tmp/speedtest.tgz "https://install.speedtest.net/app/cli/ookla-speedtest-1.2.0-linux-${arch}.tgz"; \
    echo "${sum}  /tmp/speedtest.tgz" | sha256sum -c -; \
    tar -xzf /tmp/speedtest.tgz -C /usr/local/bin speedtest

FROM debian:trixie-slim
# curl so dashdot compose files' curl healthchecks keep working; pci.ids
# names the GPUs; libcap2-bin only to set the file capability below.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl pci.ids libcap2-bin \
 && rm -rf /var/lib/apt/lists/*
COPY --from=speedtest /usr/local/bin/speedtest /usr/local/bin/speedtest
COPY --from=build /dash-comma /usr/local/bin/dash-comma
# Permitted only, not effective: Intel GPU counters work with
# cap_add: [PERFMON], and other users' GPU processes show with
# cap_add: [SYS_PTRACE], both with no root. Without them the binary still runs.
RUN setcap cap_perfmon,cap_sys_ptrace=p /usr/local/bin/dash-comma && apt-get purge -y libcap2-bin && apt-get autoremove -y

ENV DASHDOT_RUNNING_IN_DOCKER=true \
    NVIDIA_VISIBLE_DEVICES=all \
    NVIDIA_DRIVER_CAPABILITIES=utility \
    HOME=/tmp
USER 65532:65532
EXPOSE 3001
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s CMD ["dash-comma", "healthcheck"]
ENTRYPOINT ["dash-comma"]
