import { MemoryIcon } from "@phosphor-icons/react";
import { AreaChart } from "../charts/AreaChart";
import { bytes, countNames, percent } from "../format";
import type { Config, RamInfo, RamLoad } from "../types";
import { Chart, Widget } from "../Widget";
import { pick } from "./stats";

export function RamChart({ load, size, config }: { load: RamLoad[]; size: number; config: Config }) {
  const values = load.map((l) => (size ? (l / size) * 100 : 0));
  const latest = load.at(-1);
  return (
    <Chart label="Used" value={latest == null ? undefined : `${bytes(latest)} of ${bytes(size)}`}>
      <AreaChart values={values} capacity={config.ram_shown_datapoints} max={100} hue="purple" label="Used" format={percent} className="tw:grow" />
    </Chart>
  );
}

export function RamWidget({ info, load, config }: { info: RamInfo; load: RamLoad[]; config: Config }) {
  const o = config.override;
  const size = o.ram_size ?? info.size;
  const first = info.layout[0];
  const stats = pick(config.ram_label_list, {
    brand: { label: "Brand", value: o.ram_brand ?? (countNames(info.layout.map((m) => m.brand ?? "Unknown")) || undefined) },
    size: { label: "Size", value: bytes(size) },
    type: { label: "Type", value: o.ram_type ?? first?.type },
    frequency: { label: "Frequency", value: (o.ram_frequency ?? first?.frequency) ? `${o.ram_frequency ?? first?.frequency} MHz` : undefined },
  });
  return (
    <Widget title="Memory" icon={MemoryIcon} hue="purple" grow={config.ram_widget_grow} minWidth={config.ram_widget_min_width} pages={[stats]}>
      <RamChart load={load} size={size} config={config} />
    </Widget>
  );
}
