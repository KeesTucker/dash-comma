import type { ButtonHTMLAttributes, MouseEvent, ReactNode } from "react";
import { button, defaultIconSize } from "../recipes/button";
import { Spinner } from "./Spinner";

export type IconButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children" | "aria-label"> & {
  /** What the button does, read by screen readers (and shown as the tooltip). Required. */
  label: string;
  icon: ReactNode;
  variant?: "primary" | "secondary" | "outline" | "ghost" | "accent" | "danger" | "inverse";
  size?: "sm" | "md" | "lg";
  /** `rounded` matches Button; `circle` for close, overflow and floating actions. */
  shape?: "rounded" | "circle";
  /** Override the icon size (the nav bar uses md in md buttons). */
  iconSize?: "sm" | "md";
  loading?: boolean;
  /** Something new: `true` shows a dot, a number shows a count (99+ above 99). Added to the accessible name. */
  indicator?: boolean | number;
};

// Ring colour follows the surface the button sits on (--indicator-ring; TopNav sets it to moss-600).
const indicatorBase =
  "tw:pointer-events-none tw:absolute tw:box-border tw:rounded-full tw:bg-danger tw:ring-2 tw:ring-(--indicator-ring,var(--color-bg))";

function Indicator({ value }: { value: boolean | number }) {
  if (value === true)
    return <span aria-hidden className={`${indicatorBase} tw:end-1.5 tw:top-1.5 tw:size-2.5`} />;
  return (
    <span
      aria-hidden
      className={`${indicatorBase} tw:end-0 tw:top-0 tw:flex tw:h-4 tw:min-w-4 tw:items-center tw:justify-center tw:px-1 tw:font-sans tw:text-[length:var(--text-xs)] tw:leading-none tw:font-semibold tw:text-text-inverse tw:tabular-nums`}
    >
      {Number(value) > 99 ? "99+" : value}
    </span>
  );
}

/** A square, icon-only button. */
export function IconButton({ label, icon, variant, size, shape, iconSize, indicator, loading = false, type = "button", className, onClick, title, ...rest }: IconButtonProps) {
  return (
    <button
      type={type}
      aria-label={indicator === true ? `${label} (new)` : indicator ? `${label} (${indicator})` : label}
      title={title ?? label}
      aria-busy={loading || undefined}
      onClick={(e: MouseEvent<HTMLButtonElement>) => (loading ? e.preventDefault() : onClick?.(e))}
      className={button({ variant, size, shape, iconSize: iconSize ?? defaultIconSize(size), iconOnly: true, className })}
      {...rest}
    >
      {loading ? <Spinner /> : icon}
      {indicator !== undefined && indicator !== false && indicator !== 0 && <Indicator value={indicator} />}
    </button>
  );
}
