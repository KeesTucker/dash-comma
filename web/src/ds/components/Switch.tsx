import type { InputHTMLAttributes, ReactNode } from "react";
import { choice, switchThumb, switchTrack } from "../recipes/choice";

export type SwitchProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "children" | "role"> & {
  label: ReactNode;
  description?: ReactNode;
};

/** A setting that takes effect immediately (no Save button). */
export function Switch({ label, description, disabled, className = "", ...rest }: SwitchProps) {
  return (
    <label className={`${choice.root({ disabled: !!disabled })} ${className}`}>
      <span className={choice.slot}>
        <input type="checkbox" role="switch" disabled={disabled} className={choice.input} {...rest} />
        <span className={switchTrack({ disabled: !!disabled })}>
          <span className={switchThumb} />
        </span>
      </span>
      <span className={choice.text}>
        <span className={choice.label}>{label}</span>
        {description && <span className={choice.description}>{description}</span>}
      </span>
    </label>
  );
}
