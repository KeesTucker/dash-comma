import { GraphicsCardIcon } from "@phosphor-icons/react";
import { useState } from "react";
import { AreaChart } from "../charts/AreaChart";
import { bytes, percent } from "../format";
import type { Config, GpuInfo, GpuLoad } from "../types";
import { Chart, Widget } from "../Widget";
import { TopProcesses } from "./Processes";
import { pick } from "./stats";

/**
 * One GPU's charts. Intel reports each engine class (Render/3D, Video...),
 * shown as small multiples; the rest report load and memory controller use.
 * `filter` picks one chart: an engine name, "load" or "memory".
 */
export function GpuChart({ load, index, config, filter }: { load: GpuLoad[]; index: number; config: Config; filter?: string }) {
  const capacity = config.gpu_shown_datapoints;
  const engines = load.at(-1)?.layout[index]?.engines;
  const series = engines
    ? Object.keys(engines).map((name) => ({ key: name.toLowerCase(), label: name, values: load.map((l) => l.layout[index]?.engines?.[name] ?? 0) }))
    : [
        { key: "load", label: "Load", values: load.map((l) => l.layout[index]?.load ?? 0) },
        { key: "memory", label: "Memory controller", values: load.map((l) => l.layout[index]?.memory ?? 0) },
      ];
  const shown = filter ? series.filter((s) => s.key === filter || s.key.startsWith(filter)).slice(0, 1) : series;
  return (
    <div className="tw:grid tw:grow tw:grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] tw:gap-3">
      {(shown.length ? shown : series.slice(0, 1)).map((s) => {
        const latest = s.values.at(-1);
        return (
          <Chart key={s.key} label={s.label} value={latest == null ? undefined : percent(latest)}>
            <AreaChart values={s.values} capacity={capacity} max={100} hue="pink" label={s.label} format={percent} className="tw:min-h-20 tw:grow" />
          </Chart>
        );
      })}
    </div>
  );
}

export function GpuWidget({ info, load, config }: { info: GpuInfo; load: GpuLoad[]; config: Config }) {
  const [page, setPage] = useState(0);
  const o = config.override;
  const pages = info.layout.map((g, i) => {
    const memory = o.gpu_memories[i] ?? g.memory;
    return pick(config.gpu_label_list, {
      brand: { label: "Brand", value: o.gpu_brands[i] ?? g.brand },
      model: { label: "Model", value: o.gpu_models[i] ?? g.model },
      memory: { label: "Memory", value: memory ? bytes(memory * 1024 * 1024) : undefined },
    });
  });
  return (
    <Widget title="Graphics" icon={GraphicsCardIcon} hue="pink" grow={config.gpu_widget_grow} minWidth={config.gpu_widget_min_width} pages={pages} page={page} onPage={setPage}>
      {info.layout.length ? (
        <>
          <GpuChart load={load} index={page} config={config} />
          <TopProcesses processes={load.at(-1)?.layout[page]?.processes} hue="pink" />
        </>
      ) : (
        <p className="tw:m-0 tw:text-sm tw:text-text-muted">No GPUs found.</p>
      )}
    </Widget>
  );
}
