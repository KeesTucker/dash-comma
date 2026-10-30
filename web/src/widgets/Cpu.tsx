import { CpuIcon } from "@phosphor-icons/react";
import { Switch } from "../ds";
import { AreaChart } from "../charts/AreaChart";
import { percent, temp } from "../format";
import { useSetting } from "../settings";
import type { Config, CpuInfo, CpuLoad } from "../types";
import { Chart, Widget } from "../Widget";
import { pick } from "./stats";

const avg = (ns: number[]) => (ns.length ? ns.reduce((a, b) => a + b, 0) / ns.length : 0);

function tempOf(load: CpuLoad | undefined, config: Config) {
  if (!config.enable_cpu_temps || !load?.length) return undefined;
  const temps = load.map((c) => c.temp);
  return temp(config.cpu_temps_mode === "max" ? Math.max(...temps) : avg(temps), config.use_imperial);
}

export function CpuChart({ load, config, multiCore }: { load: CpuLoad[]; config: Config; multiCore: boolean }) {
  const capacity = config.cpu_shown_datapoints;
  if (multiCore) {
    const cores = load.at(-1) ?? [];
    return (
      <div className="tw:grid tw:grow tw:grid-cols-[repeat(auto-fill,minmax(7rem,1fr))] tw:gap-3">
        {cores.map((c) => (
          <Chart key={c.core} label={`Core ${c.core + 1}`} value={percent(c.load)}>
            <AreaChart values={load.map((l) => l[c.core]?.load ?? 0)} capacity={capacity} max={100} hue="blue" label={`Core ${c.core + 1}`} format={percent} className="tw:h-12" />
          </Chart>
        ))}
      </div>
    );
  }
  const values = load.map((l) => avg(l.map((c) => c.load)));
  const t = tempOf(load.at(-1), config);
  const latest = values.at(-1);
  return (
    <Chart label={t ? `Load, ${t}` : "Load"} value={latest == null ? undefined : percent(latest)}>
      <AreaChart values={values} capacity={capacity} max={100} hue="blue" label="Load" format={percent} className="tw:grow" />
    </Chart>
  );
}

export function CpuWidget({ info, load, config }: { info: CpuInfo; load: CpuLoad[]; config: Config }) {
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
    </Widget>
  );
}
