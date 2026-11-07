import { CpuIcon } from "@phosphor-icons/react";
import { Switch } from "../ds";
import { AreaChart } from "../charts/AreaChart";
import { percent, temp } from "../format";
import { useSetting } from "../settings";
import type { Config, CpuInfo, CpuLoad, ProcessLoad } from "../types";
import { Chart, Widget } from "../Widget";
import { TopProcesses } from "./Processes";
import { pick } from "./stats";

const avg = (ns: number[]) => (ns.length ? ns.reduce((a, b) => a + b, 0) / ns.length : 0);

/** The temperature series: each reading's hottest core or the average, per cpu_temps_mode. */
function temps(load: CpuLoad[], config: Config) {
  if (!config.enable_cpu_temps) return undefined;
  return load.map((l) => {
    const ts = l.map((c) => c.temp);
    return config.cpu_temps_mode === "max" ? Math.max(0, ...ts) : avg(ts);
  });
}

/**
 * Load, overall or per core, plus a temperature chart when temps are on.
 * `filter` shows just one: "load" or "temp".
 */
export function CpuChart({ load, config, multiCore, filter }: { load: CpuLoad[]; config: Config; multiCore: boolean; filter?: string }) {
  const capacity = config.cpu_shown_datapoints;
  const fmtTemp = (c: number) => temp(c, config.use_imperial);
  const t = filter === "load" ? undefined : temps(load, config);
  const tempChart = (className: string) => {
    if (!t) return null;
    const latest = t.at(-1);
    return (
      <Chart key="temp" label="Temperature" value={latest == null ? undefined : fmtTemp(latest)}>
        <AreaChart values={t} capacity={capacity} max={100} hue="blue" label="Temperature" format={fmtTemp} className={className} />
      </Chart>
    );
  };
  if (filter === "temp" && t) {
    return <div className="tw:flex tw:grow tw:flex-col">{tempChart("tw:grow")}</div>;
  }

  if (multiCore) {
    const cores = load.at(-1) ?? [];
    return (
      <div className="tw:grid tw:grow tw:grid-cols-[repeat(auto-fill,minmax(7rem,1fr))] tw:gap-3">
        {cores.map((c) => (
          <Chart key={c.core} label={`Core ${c.core + 1}`} value={percent(c.load)}>
            <AreaChart values={load.map((l) => l[c.core]?.load ?? 0)} capacity={capacity} max={100} hue="blue" label={`Core ${c.core + 1}`} format={percent} className="tw:h-12" />
          </Chart>
        ))}
        {tempChart("tw:h-12")}
      </div>
    );
  }
  const values = load.map((l) => avg(l.map((c) => c.load)));
  const latest = values.at(-1);
  return (
    <div className="tw:flex tw:grow tw:flex-col tw:gap-3 tw:sm:flex-row">
      <Chart label="Load" value={latest == null ? undefined : percent(latest)}>
        <AreaChart values={values} capacity={capacity} max={100} hue="blue" label="Load" format={percent} className="tw:grow" />
      </Chart>
      {tempChart("tw:grow")}
    </div>
  );
}

export function CpuWidget({ info, load, processes, config }: { info: CpuInfo; load: CpuLoad[]; processes: ProcessLoad[]; config: Config }) {
  const [multi, setMulti] = useSetting("multiCore", false);
  const mode = config.cpu_cores_toggle_mode;
  const multiCore = mode === "multi-core" || (mode === "toggle" && multi);
  const o = config.override;
  const freq = o.cpu_frequency ?? info.frequency;
  const stats = pick(config.cpu_label_list, {
    brand: { label: "Brand", value: o.cpu_brand ?? info.brand },
    model: { label: "Model", value: o.cpu_model ?? info.model },
    cores: { label: "Cores", value: o.cpu_cores ?? info.cores },
    threads: { label: "Threads", value: o.cpu_threads ?? info.threads },
    frequency: { label: "Frequency", value: freq ? `${freq} GHz` : undefined },
  });
  return (
    <Widget
      title="Processor"
      icon={CpuIcon}
      hue="blue"
      grow={config.cpu_widget_grow}
      minWidth={config.cpu_widget_min_width}
      pages={[stats]}
      actions={mode === "toggle" && <Switch label="Per core" checked={multi} onChange={(e) => setMulti(e.target.checked)} />}
    >
      <CpuChart load={load} config={config} multiCore={multiCore} />
      <TopProcesses processes={processes} hue="blue" />
    </Widget>
  );
}
