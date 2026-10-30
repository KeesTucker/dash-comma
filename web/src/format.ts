const nz = "en-NZ";

const num = (n: number, digits = 1) => n.toLocaleString(nz, { maximumFractionDigits: digits });

/** 1024-based, as dashdot showed sizes: "931.5 GiB". */
export function bytes(n: number): string {
  const units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
  let i = 0;
  while (Math.abs(n) >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${num(n, i === 0 ? 0 : 1)} ${units[i]}`;
}

/** A rate given in bits per second, shown in bits or bytes. */
export function speed(bitsPerSec: number, asBytes: boolean): string {
  const units = asBytes ? ["B/s", "KB/s", "MB/s", "GB/s"] : ["b/s", "Kb/s", "Mb/s", "Gb/s"];
  let n = asBytes ? bitsPerSec / 8 : bitsPerSec;
  let i = 0;
  while (n >= 1000 && i < units.length - 1) {
    n /= 1000;
    i++;
  }
  return `${num(n, n < 10 ? 2 : 1)} ${units[i]}`;
}

export const percent = (n: number) => `${num(n, 1)}%`;

export const temp = (celsius: number, imperial: boolean) =>
  imperial ? `${num(celsius * 1.8 + 32, 0)}°F` : `${num(celsius, 0)}°C`;

export function duration(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

/** "Samsung, 2x Seagate": repeated names counted. */
export function countNames(names: string[]): string {
  const counts = new Map<string, number>();
  for (const n of names) counts.set(n, (counts.get(n) ?? 0) + 1);
  return [...counts].map(([n, c]) => (c === 1 ? n : `${c}x ${n}`)).join(", ");
}
