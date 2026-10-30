import { NetworkIcon } from "@phosphor-icons/react";
import { AreaChart } from "../charts/AreaChart";
import { speed } from "../format";
import type { Config, NetworkInfo, NetworkLoad } from "../types";
import { Chart, Widget } from "../Widget";
import { pick } from "./stats";

/** Down and up as two charts, each on its own scale, never one dual-axis chart. `filter` shows just one. */
export function NetworkChart({ load, config, filter }: { load: NetworkLoad[]; config: Config; filter?: string }) {
  const fmt = (bytesPerSec: number) => speed(bytesPerSec * 8, config.network_speed_as_bytes);
  const series = [
    { key: "down", label: "Download", hue: "blue", values: load.map((l) => l.down) },
    { key: "up", label: "Upload", hue: "orange", values: load.map((l) => l.up) },
  ] as const;
  return (
    <div className="tw:flex tw:grow tw:flex-col tw:gap-3 tw:sm:flex-row">
      {series
        .filter((s) => !filter || filter === s.key)
        .map((s) => {
          const latest = s.values.at(-1);
          return (
            <Chart key={s.key} label={s.label} value={latest == null ? undefined : fmt(latest)}>
              <AreaChart values={[...s.values]} capacity={config.network_shown_datapoints} hue={s.hue} label={s.label} format={fmt} className="tw:grow" />
            </Chart>
          );
        })}
    </div>
  );
}

export function NetworkWidget({ info, load, config }: { info: NetworkInfo; load: NetworkLoad[]; config: Config }) {
  const o = config.override;
  const asBytes = config.network_speed_as_bytes;
  const down = o.network_speed_down ?? info.speedDown;
  const up = o.network_speed_up ?? info.speedUp;
  const link = o.network_interface_speed ?? info.interfaceSpeed;
  const stats = pick(config.network_label_list, {
    type: { label: "Type", value: o.network_type ?? info.type },
    speed_down: { label: "Speed test down", value: down ? speed(down, asBytes) : "Not run yet" },
    speed_up: { label: "Speed test up", value: up ? speed(up, asBytes) : "Not run yet" },
    // The link speed is in Mb/s.
    interface_speed: { label: "Link", value: link ? speed(link * 1_000_000, asBytes) : undefined },
    public_ip: { label: "Public IP", value: o.network_public_ip ?? (info.publicIp || undefined) },
  });
  return (
    <Widget title="Network" icon={NetworkIcon} hue="orange" grow={config.network_widget_grow} minWidth={config.network_widget_min_width} pages={[stats]}>
      <NetworkChart load={load} config={config} />
    </Widget>
  );
}
