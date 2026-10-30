import { hueVar, type Hue } from "./AreaChart";

type Props = {
  /** 0 to 1. */
  value: number;
  hue: Hue;
  label: string;
  valueLabel: string;
  detail?: string;
};

/** How full something is: a label and value above a bar. */
export function Meter({ value, hue, label, valueLabel, detail }: Props) {
  const v = Math.min(1, Math.max(0, value));
  return (
    <div className="tw:flex tw:flex-col tw:gap-1">
      <div className="tw:flex tw:items-baseline tw:justify-between tw:gap-3">
        <span className="tw:min-w-0 tw:truncate tw:text-sm tw:text-text">{label}</span>
        <span className="tw:shrink-0 tw:text-sm-strong tw:text-text tw:tabular-nums">{valueLabel}</span>
      </div>
      <div
        role="meter"
        aria-label={label}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(v * 100)}
        aria-valuetext={detail ? `${valueLabel}, ${detail}` : valueLabel}
        className="tw:h-2 tw:overflow-hidden tw:rounded-full tw:bg-track"
      >
        <div className="tw:h-full tw:rounded-full" style={{ width: `${v * 100}%`, background: hueVar(hue) }} />
      </div>
      {detail && <span className="tw:text-xs tw:text-text-muted">{detail}</span>}
    </div>
  );
}
