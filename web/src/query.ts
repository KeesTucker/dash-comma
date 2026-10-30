import type { Widget } from "./types";

export type Graph = Exclude<Widget, "os">;

export type SingleWidget = {
  graph: Graph;
  multiView: boolean;
  showPercentage: boolean;
  textOffset: string;
  textSize: string;
  theme?: "light" | "dark";
  color?: string;
  surface?: string;
  radius?: string;
  gap?: string;
  filter: string;
};

const sizeRe = /^\d+(\.\d+)?(px|rem|em|%|vh|vw|ch|ex)$/i;
const numberRe = /^\d+(\.\d+)?$/;
const colorRe = /^([A-Fa-f0-9]{3,4}|[A-Fa-f0-9]{6}|[A-Fa-f0-9]{8})$/;

const size = (v: string | null) => (v == null ? undefined : sizeRe.test(v) ? v : numberRe.test(v) ? `${v}px` : undefined);
const color = (v: string | null) => (v != null && colorRe.test(v) ? `#${v}` : undefined);

/**
 * dashdot's embed mode: `?graph=cpu` shows one chart filling the page, for
 * iframes in other dashboards. The parameters are the same as dashdot's.
 */
export function readSingleWidget(search = window.location.search): SingleWidget | undefined {
  const q = new URLSearchParams(search);
  const graph = q.get("graph");
  if (graph == null) return undefined;
  const theme = q.get("theme");
  return {
    graph: graph as Graph,
    multiView: q.get("multiView") === "true",
    showPercentage: q.get("showPercentage") === "true",
    textOffset: size(q.get("textOffset")) ?? "24px",
    textSize: size(q.get("textSize")) ?? "16px",
    theme: theme === "light" || theme === "dark" ? theme : undefined,
    color: color(q.get("color")),
    surface: color(q.get("surface")),
    radius: size(q.get("innerRadius")),
    gap: size(q.get("gap")),
    filter: (q.get("filter") ?? "").toLowerCase(),
  };
}
