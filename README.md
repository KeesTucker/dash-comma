# dash-comma

A server dashboard. It's [dashdot](https://github.com/MauriceNino/dashdot)
rewritten in Go: the same API, the same `DASHDOT_*` settings and the same
embeddable charts, with its own stats collection and a new UI.

- **Swap the image and go.** Point an existing dashdot compose file at
  `ghcr.io/keestucker/dash-comma` and it runs: port 3001, the `/mnt/host`
  mount, every `DASHDOT_*` variable, the `/info`, `/config` and `/load/*`
  routes, the socket.io events and the `?graph=` embed links all carry over.
- **One image for every GPU.** NVIDIA through NVML, AMD through amdgpu's
  sysfs, and Intel (i915 and xe) through the GPU's perf counters, with no
  intel_gpu_top. dashdot needed a separate image per vendor.
- **Reads the host itself.** Load, memory, disks, network and temperatures
  come straight from procfs and sysfs, not from a library shelling out to
  `df`, `lsblk` and `dmidecode`. It needs no privileged mode and no root.
- **Pools count once.** A mergerfs or ZFS pool in `DASHDOT_FS_VIRTUAL_MOUNTS`
  takes in the disks under it, and shows in the storage chart. Btrfs
  subvolumes aren't counted once per mount.

## Run it

```yaml
services:
  dash:
    image: ghcr.io/keestucker/dash-comma:1.0.0 # x-release-please-version
    restart: unless-stopped
    ports:
      - 3001:3001
    volumes:
      - /:/mnt/host:ro
    # Intel GPUs: lets dash-comma read the engine counters.
    cap_add:
      - PERFMON
    environment:
      DASHDOT_WIDGET_LIST: os,cpu,storage,ram,network,gpu
```

NVIDIA cards also need the
[NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/)
and a GPU reservation (`deploy.resources.reservations.devices`). AMD cards
need nothing extra.

dashdot's `privileged: true` isn't needed. The container runs as a non-root
user (65532); a `user:` in your compose file still wins.

## Settings

Every variable from [dashdot's configuration docs](https://getdashdot.com/docs/configuration)
works the same way, with the same defaults. Two behave a little better:

- `DASHDOT_FS_VIRTUAL_MOUNTS` names a pool by its mount source (for mergerfs,
  its `fsname`). The disks under it are found from mergerfs's branches or the
  ZFS pool's members, so they no longer need hiding with
  `DASHDOT_FS_DEVICE_FILTER`. The pool is read even though `fuse.mergerfs` is
  in the default `DASHDOT_FS_TYPE_FILTER`.
- RAM brand, type and speed come from SMBIOS, which only root can read. In a
  non-root container set them with `DASHDOT_OVERRIDE_RAM_*`, as before.

## Embedding

`/?graph=cpu|ram|storage|network|gpu` shows one chart filling the page, with
dashdot's parameters: `multiView`, `showPercentage`, `textSize`, `textOffset`,
`theme`, `color`, `surface`, `innerRadius`, `gap` and `filter`.

## Building

```sh
(cd web && bun install && bun run build)
go build .
```

Or `docker build .`. The NVIDIA support uses cgo, so a local build needs a C
compiler.

## Releases

Commits follow [Conventional Commits](https://www.conventionalcommits.org).
release-please keeps a release PR open on `main`; merging it tags `vX.Y.Z` and
publishes the image as `X.Y.Z`, `X.Y`, `X` and `latest`, for amd64 and arm64.
Use `X` to take every release short of a breaking one.
