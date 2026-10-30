import { useEffect, useState } from "react";
import { io, type Socket } from "socket.io-client";
import type { CpuLoad, GpuLoad, NetworkLoad, RamLoad, ServerInfo, StorageLoad } from "./types";

// The socket sits next to the page, so a DASHDOT_ROUTING_PATH of /dash
// finds /dash/socket. `bun run dev` talks to a local server on :3001.
function connect(): Socket {
  const base = new URL(".", window.location.href);
  const origin = import.meta.env.DEV ? "http://localhost:3001" : base.origin;
  return io(origin, { path: `${base.pathname.replace(/\/$/, "")}/socket` });
}

/** old with next added, keeping the last max. */
const append = <T,>(old: T[], next: T, max: number) => [...old, next].slice(-Math.max(1, max));

export type PageData = {
  connected: boolean;
  error: boolean;
  info?: ServerInfo;
  cpu: CpuLoad[];
  ram: RamLoad[];
  network: NetworkLoad[];
  gpu: GpuLoad[];
  storage?: StorageLoad;
};

/** Live data from the socket: static info, plus a capped history of each load. */
export function usePageData(): PageData {
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState(false);
  const [info, setInfo] = useState<ServerInfo>();
  const [cpu, setCpu] = useState<CpuLoad[]>([]);
  const [ram, setRam] = useState<RamLoad[]>([]);
  const [network, setNetwork] = useState<NetworkLoad[]>([]);
  const [gpu, setGpu] = useState<GpuLoad[]>([]);
  const [storage, setStorage] = useState<StorageLoad>();

  useEffect(() => {
    const socket = connect();
    // Caps come from the config, which arrives first on the same socket.
    let caps = { cpu: 20, ram: 20, network: 20, gpu: 20 };

    socket.on("connect", () => {
      setConnected(true);
      setError(false);
    });
    socket.on("connect_error", () => setError(true));
    socket.on("disconnect", () => setConnected(false));
    socket.on("static-info", (data: ServerInfo) => {
      const c = data.config;
      caps = { cpu: c.cpu_shown_datapoints, ram: c.ram_shown_datapoints, network: c.network_shown_datapoints, gpu: c.gpu_shown_datapoints };
      setInfo(data);
    });
    socket.on("cpu-load", (d: CpuLoad) => setCpu((old) => append(old, d, caps.cpu)));
    socket.on("ram-load", (d: RamLoad) => setRam((old) => append(old, d, caps.ram)));
    socket.on("network-load", (d: NetworkLoad) => setNetwork((old) => append(old, d, caps.network)));
    socket.on("gpu-load", (d: GpuLoad) => setGpu((old) => append(old, d, caps.gpu)));
    socket.on("storage-load", (d: StorageLoad) => setStorage(d));

    return () => {
      socket.close();
    };
  }, []);

  return { connected, error, info, cpu, ram, network, gpu, storage };
}
