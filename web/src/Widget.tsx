import { CaretLeftIcon, CaretRightIcon, type Icon } from "@phosphor-icons/react";
import type { CSSProperties, ReactNode } from "react";
import { Card, crayonTile, IconButton, iconTile, sectionHeader } from "./ds";
import type { Hue } from "./charts/AreaChart";

export type Stat = { label: string; value: ReactNode };

/** Readings in a grid: a quiet label over each value. */
export function StatGrid({ stats }: { stats: Stat[] }) {
  if (!stats.length) return null;
  return (
    <dl className="tw:m-0 tw:grid tw:grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] tw:gap-x-4 tw:gap-y-3">
      {stats.map((s) => (
        <div key={s.label} className="tw:flex tw:min-w-0 tw:flex-col">
          <dt className="tw:text-xs tw:text-text-muted">{s.label}</dt>
          <dd className="tw:m-0 tw:truncate tw:text-sm-strong tw:text-text tw:tabular-nums">{s.value ?? "Unknown"}</dd>
        </div>
      ))}
    </dl>
  );
}

type Props = {
  title: string;
  icon: Icon;
  hue: Hue;
  /** The widget's DASHDOT_*_WIDGET_GROW and _MIN_WIDTH. */
  grow: number;
  minWidth: number;
  /** One page of stats per device; paging shows when there's more than one. */
  pages: Stat[][];
  page?: number;
  onPage?: (page: number) => void;
  /** Settings shown in the header, like the per-core switch. */
  actions?: ReactNode;
  children?: ReactNode;
};

export function Widget({ title, icon: TileIcon, hue, grow, minWidth, pages, page = 0, onPage, actions, children }: Props) {
  const count = pages.length;
  // grow and min width are the user's config, so they can't be tokens.
  const style: CSSProperties = { flexGrow: grow, flexBasis: `${minWidth}px`, minWidth: `min(${minWidth}px, 100%)` };
  return (
    <Card padding="md" className="tw:box-border tw:flex tw:flex-col tw:gap-4" style={style}>
      <div className={sectionHeader.root}>
        <div className={sectionHeader.start}>
          <span className={`${iconTile({ size: "md" })} ${crayonTile[hue]}`} aria-hidden>
            <TileIcon />
          </span>
          <h2 className={sectionHeader.title}>{title}</h2>
        </div>
        <div className={sectionHeader.actions}>
          {actions}
          {count > 1 && onPage && (
            <>
              <IconButton label={`Previous ${title.toLowerCase()}`} icon={<CaretLeftIcon />} variant="ghost" size="sm" disabled={page === 0} onClick={() => onPage(page - 1)} />
              <span className="tw:text-xs tw:text-text-muted tw:tabular-nums">
                {page + 1} of {count}
              </span>
              <IconButton label={`Next ${title.toLowerCase()}`} icon={<CaretRightIcon />} variant="ghost" size="sm" disabled={page === count - 1} onClick={() => onPage(page + 1)} />
            </>
          )}
        </div>
      </div>
      <StatGrid stats={pages[page] ?? []} />
      {children && <div className="tw:flex tw:min-h-40 tw:grow tw:flex-col tw:gap-3">{children}</div>}
    </Card>
  );
}

/** A chart with its current value as a direct label above it. */
export function Chart({ label, value, children }: { label: string; value?: string; children: ReactNode }) {
  return (
    <div className="tw:flex tw:min-h-0 tw:min-w-0 tw:grow tw:flex-col tw:gap-1">
      <div className="tw:flex tw:items-baseline tw:justify-between tw:gap-2">
        <span className="tw:text-xs tw:text-text-muted">{label}</span>
        {value && <span className="tw:text-sm-strong tw:text-text tw:tabular-nums">{value}</span>}
      </div>
      {children}
    </div>
  );
}
