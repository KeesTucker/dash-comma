import { cva } from "class-variance-authority";

export type Tone = "info" | "success" | "warning" | "danger";

// ---------- Alert: a status message that stays on the page
export const alert = cva(
  "tw:box-border tw:flex tw:w-full tw:items-start tw:gap-3 tw:rounded-xl tw:p-4 tw:font-sans tw:inset-ring tw:[&>svg]:size-(--icon-md) tw:[&>svg]:shrink-0",
  {
    variants: {
      tone: {
        info: "tw:bg-info-bg tw:inset-ring-info-border tw:[&>svg]:text-info-icon",
        success: "tw:bg-success-bg tw:inset-ring-success-border tw:[&>svg]:text-success-icon",
        warning: "tw:bg-warning-bg tw:inset-ring-warning-border tw:[&>svg]:text-warning-icon",
        danger: "tw:bg-danger-bg tw:inset-ring-danger-border tw:[&>svg]:text-danger-icon",
      },
    },
    defaultVariants: { tone: "info" },
  },
);

export const alertTitle = cva("tw:m-0 tw:text-sm-strong", {
  variants: {
    tone: { info: "tw:text-info-text", success: "tw:text-success-text", warning: "tw:text-warning-text", danger: "tw:text-danger-text" },
  },
  defaultVariants: { tone: "info" },
});

export const alertParts = {
  body: "tw:flex tw:min-w-0 tw:flex-1 tw:flex-col tw:gap-1",
  text: "tw:m-0 tw:text-sm tw:text-text",
  actions: "tw:mt-2 tw:flex tw:flex-wrap tw:gap-2",
  // collapse and dismiss buttons: a box one title line tall, so the 32px buttons centre on the title
  controls: "tw:-me-2 tw:flex tw:h-(--leading-sm) tw:items-center",
};

// Alert actions in the alert's own colour. Added to the Button recipe with variant: null (sizes, focus, dent stay).
export const alertAction = cva("", {
  variants: {
    kind: { primary: "", secondary: "tw:bg-transparent tw:inset-ring tw:enabled:hover:bg-bg tw:enabled:active:bg-bg" },
    tone: { info: "", success: "", warning: "", danger: "" },
  },
  compoundVariants: [
    { kind: "primary", tone: "info", className: "tw:bg-primary tw:text-text-on-primary tw:enabled:hover:bg-primary-hover tw:enabled:active:bg-primary-pressed" },
    // tick-600 fails 4.5:1 with white text, so success starts at 700
    { kind: "primary", tone: "success", className: "tw:bg-mode-tick-700 tw:text-text-on-success tw:enabled:hover:bg-mode-tick-800 tw:enabled:active:bg-mode-tick-800" },
    { kind: "primary", tone: "warning", className: "tw:bg-mode-cone-700 tw:text-text-on-warning tw:enabled:hover:bg-mode-cone-800 tw:enabled:active:bg-mode-cone-800" },
    { kind: "primary", tone: "danger", className: "tw:bg-danger tw:text-text-on-danger tw:enabled:hover:bg-danger-hover tw:enabled:active:bg-danger-pressed" },
    { kind: "secondary", tone: "info", className: "tw:text-info-text tw:inset-ring-info-border" },
    { kind: "secondary", tone: "success", className: "tw:text-success-text tw:inset-ring-success-border" },
    { kind: "secondary", tone: "warning", className: "tw:text-warning-text tw:inset-ring-warning-border" },
    { kind: "secondary", tone: "danger", className: "tw:text-danger-text tw:inset-ring-danger-border" },
  ],
  defaultVariants: { kind: "primary", tone: "info" },
});
