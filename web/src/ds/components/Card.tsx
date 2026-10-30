import type { HTMLAttributes, ReactNode } from "react";
import { card } from "../recipes/card";

export type CardProps = Omit<HTMLAttributes<HTMLDivElement>, "children"> & {
  padding?: "none" | "sm" | "md" | "lg";
  children: ReactNode;
};

/** A surface that groups one thing. */
export function Card({ padding = "md", className = "", children, ...rest }: CardProps) {
  return (
    <div className={`${card({ padding })} ${className}`} {...rest}>
      {children}
    </div>
  );
}
