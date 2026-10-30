import type { CSSProperties } from "react";
import type { SingleWidget as Query } from "./query";
import type { PageData } from "./usePageData";
import { CpuChart } from "./widgets/Cpu";
import { GpuChart } from "./widgets/Gpu";
import { NetworkChart } from "./widgets/Network";
import { RamChart } from "./widgets/Ram";
import { StorageChart } from "./widgets/Storage";

/**
 * One chart filling the page, for embedding in another dashboard's iframe.
 * ?color, ?surface, ?innerRadius, ?gap, ?textSize and ?textOffset are the
 * embedder's own values, so they go in as inline styles.
 */
export function SingleWidget({ data, query }: { data: PageData; query: Query }) {
  const { info } = data;
  if (!info) return null;
  const config = info.config;

  const style = {
    "--chart-color": query.color,
    background: query.surface ?? "transparent",
    borderRadius: query.radius,
    gap: query.gap,
    fontSize: query.textSize,
    padding: query.textOffset,
  } as CSSProperties;

  let chart = null;
  switch (query.graph) {
    case "cpu":
      chart = <CpuChart load={data.cpu} config={config} multiCore={query.multiView} />;
      break;
    case "ram":
      chart = <RamChart load={data.ram} size={config.override.ram_size ?? info.ram.size} config={config} />;
      break;
    case "storage":
      chart = <StorageChart data={info.storage} load={data.storage} config={config} split={query.multiView} />;
      break;
    case "network":
      chart = <NetworkChart load={data.network} config={config} filter={query.filter || undefined} />;
      break;
    case "gpu":
      chart = <GpuChart load={data.gpu} index={0} config={config} filter={query.filter || undefined} />;
      break;
  }
  return (
    <div className="tw:box-border tw:flex tw:h-screen tw:w-screen tw:flex-col tw:overflow-hidden" style={style}>
      {chart}
    </div>
  );
}
