import { useState } from "react";

export type Slice = { key: string; label: string; value: number; colour: string; detail?: string };

/** Slice colours in a fixed order, checked as a set for colour-blind separation in both modes. */
export const sliceColours = ["blue", "orange", "purple", "teal", "coral"].map((h) => `var(--color-crayon-${h}-500)`);
export const otherColour = "var(--color-text-subtle)";
export const freeColour = "var(--color-track)";

const R = 50;
const WIDTH = 14;

function arc(start: number, end: number) {
  const r = R - WIDTH / 2;
  const pt = (a: number) => [R + r * Math.sin(a), R - r * Math.cos(a)];
  const [x1, y1] = pt(start);
  const [x2, y2] = pt(end);
  return `M${x1},${y1}A${r},${r} 0 ${end - start > Math.PI ? 1 : 0} 1 ${x2},${y2}`;
}

/**
 * Part of a whole at a glance: slices clockwise from the top, each with a
 * thin surface gap, the total in the middle and a legend beside it. Hover
 * or focus a slice (or its legend row) for its value.
 */
export function Donut({ slices, centre, centreLabel, format }: { slices: Slice[]; centre: string; centreLabel: string; format: (v: number) => string }) {
  const [active, setActive] = useState<string | null>(null);
  const total = slices.reduce((a, s) => a + Math.max(0, s.value), 0);
  const gap = slices.filter((s) => s.value > 0).length > 1 ? 0.03 : 0;

  const arcs = slices.reduce<(Slice & { start: number; end: number })[]>((out, s) => {
    const from = out.at(-1)?.end ?? 0;
    const sweep = total ? (Math.max(0, s.value) / total) * Math.PI * 2 : 0;
    return [...out, { ...s, start: from, end: from + sweep }];
  }, []);
  const shown = slices.find((s) => s.key === active);

  return (
    <div className="tw:flex tw:flex-wrap tw:items-center tw:gap-6">
      <div className="tw:relative tw:size-40 tw:shrink-0">
        <svg viewBox={`0 0 ${R * 2} ${R * 2}`} className="tw:size-full" role="img" aria-label={`${centreLabel}: ${centre}`}>
          {arcs.map((a) =>
            a.end - a.start <= gap ? null : a.end - a.start >= Math.PI * 2 - 0.001 ? (
              <circle key={a.key} cx={R} cy={R} r={R - WIDTH / 2} fill="none" stroke={a.colour} strokeWidth={WIDTH} />
            ) : (
              <path
                key={a.key}
                d={arc(a.start + gap / 2, a.end - gap / 2)}
                fill="none"
                stroke={a.colour}
                strokeWidth={active === a.key ? WIDTH + 2 : WIDTH}
                onPointerEnter={() => setActive(a.key)}
                onPointerLeave={() => setActive(null)}
              />
            ),
          )}
        </svg>
        <div className="tw:pointer-events-none tw:absolute tw:inset-0 tw:flex tw:flex-col tw:items-center tw:justify-center tw:text-center">
          {shown ? (
            <>
              <span className="tw:text-sm-strong tw:text-text tw:tabular-nums">{format(shown.value)}</span>
              <span className="tw:max-w-24 tw:truncate tw:text-xs tw:text-text-muted">{shown.label}</span>
            </>
          ) : (
            <>
              <span className="tw:title-lg tw:text-text">{centre}</span>
              <span className="tw:text-xs tw:text-text-muted">{centreLabel}</span>
            </>
          )}
        </div>
      </div>
      <ul className="tw:m-0 tw:flex tw:min-w-0 tw:grow tw:list-none tw:flex-col tw:gap-2 tw:p-0">
        {slices.map((s) => (
          <li
            key={s.key}
            tabIndex={0}
            className="tw:flex tw:items-center tw:gap-2 tw:rounded-sm tw:outline-none tw:focus-visible:ring-2 tw:focus-visible:ring-focus-ring"
            onPointerEnter={() => setActive(s.key)}
            onPointerLeave={() => setActive(null)}
            onFocus={() => setActive(s.key)}
            onBlur={() => setActive(null)}
          >
            <span aria-hidden className="tw:size-3 tw:shrink-0 tw:rounded-sm" style={{ background: s.colour }} />
            <span className="tw:flex tw:min-w-0 tw:grow tw:flex-col">
              <span className="tw:truncate tw:text-sm tw:text-text">{s.label}</span>
              {s.detail && <span className="tw:truncate tw:text-xs tw:text-text-muted">{s.detail}</span>}
            </span>
            <span className="tw:shrink-0 tw:text-sm-strong tw:text-text tw:tabular-nums">{format(s.value)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
