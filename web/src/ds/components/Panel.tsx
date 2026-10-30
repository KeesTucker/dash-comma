import type { HTMLAttributes, ReactNode } from "react";
import { panel } from "../recipes/overlay";

export type PanelProps = Omit<HTMLAttributes<HTMLElement>, "children"> & {
  /** section (default), main for the page's main panel, aside, div. */
  as?: "section" | "main" | "aside" | "div";
  padding?: "none" | "md" | "lg";
  /** scroll (default): the panel scrolls its own content. visible: it grows with its content. */
  overflow?: "scroll" | "visible";
  children: ReactNode;
};

/** The white container a page is built in. Sits on `surface`. */
export function Panel({ as: Tag = "section", padding, overflow, className = "", children, ...rest }: PanelProps) {
  return (
    <Tag className={`${panel({ padding, overflow })} ${className}`} {...rest}>
      {children}
    </Tag>
  );
}
