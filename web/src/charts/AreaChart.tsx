import { useId, useLayoutEffect, useRef, useState, type KeyboardEvent, type PointerEvent } from "react";

export type Hue = "blue" | "purple" | "green" | "orange" | "pink" | "teal";

/** The series colour. A single-widget embed's ?color= overrides it with --chart-color. */
export const hueVar = (hue: Hue) => `var(--chart-color, var(--color-crayon-${hue}-500))`;

function useSize<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const [size, setSize] = useState({ width: 0, height: 0 });
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const ro = new ResizeObserver(([e]) => setSize({ width: e.contentRect.width, height: e.contentRect.height }));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  return [ref, size] as const;
}

/** A clean top for the y scale: 1, 2 or 5 times a power of ten. */
function niceMax(v: number) {
  if (v <= 0) return 1;
  const p = 10 ** Math.floor(Math.log10(v));
  return [1, 2, 5, 10].map((m) => m * p).find((m) => m >= v)!;
}

type Props = {
  values: number[];
  /** How many points the chart holds; new ones enter at the right. */
  capacity: number;
  /** Fixed top of the scale (100 for a percentage); otherwise it fits the data. */
  max?: number;
  hue: Hue;
  /** What the series is, for the tooltip and screen readers: "Load". */
  label: string;
  format: (v: number) => string;
  className?: string;
};

/**
 * One series over time: a 2px line over a 10% wash, newest at the right. A
 * crosshair snaps to the nearest reading on hover, and the arrow keys step
 * through readings when focused.
 */
export function AreaChart({ values, capacity, max, hue, label, format, className = "" }: Props) {
  const [ref, { width, height }] = useSize<HTMLDivElement>();
  const [active, setActive] = useState<number | null>(null);
  const tooltipId = useId();

  const slots = Math.max(capacity, values.length, 2);
  const top = max ?? niceMax(Math.max(0, ...values) * 1.1);
  const pad = 4; // room for the end dot's ring, so it isn't clipped
  const x = (i: number) => ((slots - values.length + i) / (slots - 1)) * width;
  const y = (v: number) => pad + (1 - Math.min(Math.max(v, 0), top) / top) * (height - pad * 2);

  const line = values.map((v, i) => `${i ? "L" : "M"}${x(i).toFixed(1)},${y(v).toFixed(1)}`).join("");
  const area = values.length > 1 ? `${line}L${x(values.length - 1).toFixed(1)},${height}L${x(0).toFixed(1)},${height}Z` : "";

  const pick = (e: PointerEvent<HTMLDivElement>) => {
    if (!values.length || width === 0) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const slot = Math.round(((e.clientX - rect.left) / width) * (slots - 1));
    setActive(Math.min(values.length - 1, Math.max(0, slot - (slots - values.length))));
  };

  const step = (e: KeyboardEvent<HTMLDivElement>) => {
    if (!values.length) return;
    const last = values.length - 1;
    const at = active ?? last;
    if (e.key === "ArrowLeft") setActive(Math.max(0, at - 1));
    else if (e.key === "ArrowRight") setActive(Math.min(last, at + 1));
    else return;
    e.preventDefault();
  };

  const shown = active ?? null;
  const latest = values.at(-1);
  const colour = hueVar(hue);

  return (
    <div
      ref={ref}
      tabIndex={0}
      role="img"
      aria-label={latest == null ? `${label}: waiting for data` : `${label}: ${format(latest)}`}
      aria-describedby={shown != null ? tooltipId : undefined}
      className={`tw:relative tw:min-h-16 tw:outline-none tw:focus-visible:ring-2 tw:focus-visible:ring-focus-ring tw:rounded-md ${className}`}
      onPointerMove={pick}
      onPointerLeave={() => setActive(null)}
      onFocus={() => values.length && setActive(values.length - 1)}
      onBlur={() => setActive(null)}
      onKeyDown={step}
    >
      {width > 0 && height > 0 && (
        <svg width={width} height={height} className="tw:absolute tw:inset-0 tw:overflow-visible" aria-hidden>
          <line x1={0} x2={width} y1={height - 0.5} y2={height - 0.5} className="tw:stroke-border" strokeWidth={1} />
          {area && <path d={area} fill={colour} fillOpacity={0.1} />}
          {values.length > 1 && <path d={line} fill="none" stroke={colour} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />}
          {shown != null && (
            <>
              <line x1={x(shown)} x2={x(shown)} y1={0} y2={height} className="tw:stroke-border-strong" strokeWidth={1} />
              <circle cx={x(shown)} cy={y(values[shown])} r={4} fill={colour} className="tw:stroke-surface" strokeWidth={2} />
            </>
          )}
        </svg>
      )}
      {shown != null && (
        <div
          id={tooltipId}
          role="tooltip"
          className="tw:pointer-events-none tw:absolute tw:top-0 tw:z-(--z-tooltip) tw:flex tw:flex-col tw:rounded-md tw:bg-surface-raised tw:px-2 tw:py-1 tw:shadow-lift"
          style={x(shown) > width / 2 ? { right: width - x(shown) + 8 } : { left: x(shown) + 8 }}
        >
          <span className="tw:text-sm-strong tw:text-text tw:tabular-nums">{format(values[shown])}</span>
          <span className="tw:text-xs tw:text-text-muted">
            {label}
            {shown < values.length - 1 && `, ${values.length - 1 - shown} ago`}
          </span>
        </div>
      )}
    </div>
  );
}
