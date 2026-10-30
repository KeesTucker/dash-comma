import { HardDrivesIcon } from "@phosphor-icons/react";
import { useMemo, useState } from "react";
import { Switch } from "../ds";
import { Donut, freeColour, otherColour, sliceColours, type Slice } from "../charts/Donut";
import { Meter } from "../charts/Meter";
import { bytes, countNames, percent } from "../format";
import { useSetting } from "../settings";
import type { Config, StorageInfo, StorageLoad } from "../types";
import { Widget } from "../Widget";
import { pick } from "./stats";

/** The layout with DASHDOT_OVERRIDE_STORAGE_* applied by device. */
function withOverrides(data: StorageInfo, config: Config): StorageInfo {
  const o = config.override;
  return data.map((e) => {
    const sized = e.disks.find((d) => o.storage_sizes[d.device] != null);
    return {
      ...e,
      size: sized ? o.storage_sizes[sized.device] : e.size,
      disks: e.disks.map((d) => ({ ...d, brand: o.storage_brands[d.device] ?? d.brand, type: o.storage_types[d.device] ?? d.type })),
    };
  });
}

type Entry = StorageInfo[number];

// A pool's first disk is the pool itself (its brand is the
// DASHDOT_OVERRIDE_STORAGE_BRANDS name); the disks under it follow.
const members = (e: Entry) => (e.virtual ? e.disks.slice(1) : e.disks);

function entryName(e: Entry) {
  if (e.virtual) return e.disks[0]?.brand ?? "Pool";
  if (e.raidLabel) return `${e.raidLabel} (RAID ${e.raidType === 0 ? 0 : 1})`;
  return countNames(e.disks.map((d) => d.brand));
}

function entryDetail(e: Entry) {
  if (!e.virtual) return countNames(e.disks.map((d) => d.type));
  const m = members(e);
  return m.length ? `Pool of ${countNames(m.map((d) => d.brand))}` : "Pool";
}

/**
 * Where the space goes: a donut of what each drive or pool uses, plus what's
 * free, or a bar per drive. A pool counts once, with the disks under it.
 * -1 means no mount was found for an entry.
 */
export function StorageChart({ data, load, config, split }: { data: StorageInfo; load?: StorageLoad; config: Config; split: boolean }) {
  const layout = useMemo(() => withOverrides(data, config), [data, config]);
  if (!load) return <p className="tw:m-0 tw:text-sm tw:text-text-muted">Measuring…</p>;

  if (split) {
    return (
      <div className="tw:flex tw:flex-col tw:gap-4">
        {layout.map((e, i) => {
          const used = load[i] ?? -1;
          if (used < 0) return <Meter key={i} value={0} hue="green" label={entryName(e)} valueLabel="No mount found" />;
          return <Meter key={i} value={used / e.size} hue="green" label={entryName(e)} valueLabel={percent((used / e.size) * 100)} detail={`${bytes(used)} of ${bytes(e.size)}, ${entryDetail(e)}`} />;
        })}
      </div>
    );
  }

  const counted = layout.map((e, i) => ({ e, i, used: load[i] ?? -1 })).filter((x) => x.used >= 0);
  const size = counted.reduce((a, x) => a + x.e.size, 0);
  const used = counted.reduce((a, x) => a + x.used, 0);
  if (!size) return <p className="tw:m-0 tw:text-sm tw:text-text-muted">No mounts found.</p>;

  // Colour follows the entry's place in the layout, so it doesn't change
  // when another drive fills up. Past five, the rest share "Other".
  const slices: Slice[] = counted
    .filter((x) => x.i < sliceColours.length)
    .map((x) => ({ key: String(x.i), label: entryName(x.e), detail: entryDetail(x.e), value: x.used, colour: sliceColours[x.i] }));
  const rest = counted.filter((x) => x.i >= sliceColours.length);
  if (rest.length) slices.push({ key: "other", label: `${rest.length} more`, value: rest.reduce((a, x) => a + x.used, 0), colour: otherColour });
  slices.push({ key: "free", label: "Free", value: Math.max(0, size - used), colour: freeColour });

  return <Donut slices={slices} centre={percent((used / size) * 100)} centreLabel={`of ${bytes(size)} used`} format={bytes} />;
}

export function StorageWidget({ data, load, config }: { data: StorageInfo; load?: StorageLoad; config: Config }) {
  const [split, setSplit] = useSetting("splitStorage", false);
  const [page, setPage] = useState(0);
  const layout = useMemo(() => withOverrides(data, config), [data, config]);
  const per = Math.max(1, config.storage_widget_items_per_page);

  const pages = [];
  for (let i = 0; i < layout.length; i += per) {
    pages.push(
      layout.slice(i, i + per).flatMap((e, j) => {
        const drive = e.virtual ? entryName(e) : `Drive ${i + j + 1}`;
        return pick(config.storage_label_list, {
          brand: { label: drive, value: e.virtual ? entryDetail(e) : entryName(e) },
          size: { label: `${drive} size`, value: bytes(e.size) },
          type: { label: `${drive} type`, value: countNames(members(e).map((d) => d.type)) || "Pool" },
          raid: { label: `${drive} RAID`, value: e.raidLabel ? `${e.raidLabel}, RAID ${e.raidType === 0 ? 0 : 1}` : "None" },
        });
      }),
    );
  }

  return (
    <Widget
      title="Storage"
      icon={HardDrivesIcon}
      hue="green"
      grow={config.storage_widget_grow}
      minWidth={config.storage_widget_min_width}
      pages={pages.length ? pages : [[]]}
      page={Math.min(page, Math.max(0, pages.length - 1))}
      onPage={setPage}
      actions={layout.length > 1 && <Switch label="Per drive" checked={split} onChange={(e) => setSplit(e.target.checked)} />}
    >
      <StorageChart data={data} load={load} config={config} split={split} />
    </Widget>
  );
}
