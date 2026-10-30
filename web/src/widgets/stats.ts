import type { Stat } from "../Widget";

/** The stats a DASHDOT_*_LABEL_LIST asks for, in its order. */
export function pick<K extends string>(list: K[], all: Record<K, Stat>): Stat[] {
  return list.filter((k) => k in all).map((k) => all[k]);
}
