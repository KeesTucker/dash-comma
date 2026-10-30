import { Meter } from "../charts/Meter";
import type { Hue } from "../charts/AreaChart";
import { percent } from "../format";
import type { ProcessLoad } from "../types";

/** The busiest processes, each as a bar. Shows nothing while all are idle. */
export function TopProcesses({ processes, hue }: { processes?: ProcessLoad[]; hue: Hue }) {
  if (!processes?.length) return null;
  return (
    <section className="tw:flex tw:flex-col tw:gap-2" aria-label="Top processes">
      <h3 className="tw:m-0 tw:text-xs tw:font-normal tw:text-text-muted">Top processes</h3>
      {processes.map((p) => (
        <Meter key={p.pid} value={p.load / 100} hue={hue} label={p.name || `PID ${p.pid}`} valueLabel={percent(p.load)} />
      ))}
    </section>
  );
}
