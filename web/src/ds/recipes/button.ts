// Button recipe. Tokens only: no hex, no arbitrary px.
// Recipes are framework-neutral class lists, so an Ember component can reuse them later.
import { cva, type VariantProps } from "class-variance-authority";

export const button = cva(
  [
    // reset (no preflight)
    "tw:m-0 tw:border-0 tw:box-border tw:appearance-none tw:no-underline tw:cursor-pointer tw:select-none",
    // layout
    "tw:relative tw:inline-flex tw:shrink-0 tw:items-center tw:justify-center tw:gap-2 tw:whitespace-nowrap",
    // shape and type
    "tw:font-sans tw:font-medium",
    // icons
    "tw:[&_svg]:shrink-0",
    // motion
    "tw:transition-[color,background-color,box-shadow] tw:duration-(--duration-dash) tw:ease-standard",
    // pressed: pushed into the page (colour step per variant below)
    "tw:enabled:active:inset-shadow-dent",
    // focus (keyboard only)
    "tw:outline-hidden tw:focus-visible:shadow-focus",
    // disabled
    "tw:disabled:opacity-(--opacity-disabled) tw:disabled:cursor-not-allowed",
    // loading
    "tw:aria-busy:cursor-progress",
    // hit area: at least target-min, centred on the button
    "tw:before:absolute tw:before:top-1/2 tw:before:left-1/2 tw:before:-translate-1/2 tw:before:size-full tw:before:min-h-(--target-min) tw:before:min-w-(--target-min)",
  ],
  {
    variants: {
      variant: {
        primary: "tw:bg-primary tw:text-text-on-primary tw:enabled:hover:bg-primary-hover tw:enabled:active:bg-primary-pressed",
        secondary: "tw:bg-mode-moss-100 tw:text-mode-moss-800 tw:enabled:hover:bg-mode-moss-200 tw:enabled:active:bg-mode-moss-300",
        outline:
          "tw:bg-bg tw:text-text tw:inset-ring tw:inset-ring-border-strong tw:enabled:hover:bg-surface-outline-hover tw:enabled:hover:inset-ring-border-hover tw:enabled:active:bg-surface-outline-pressed tw:enabled:active:inset-ring-border-hover",
        ghost: "tw:bg-transparent tw:text-mode-moss-700 tw:enabled:hover:bg-mode-moss-50 tw:enabled:active:bg-mode-moss-100",
        accent: "tw:bg-mode-bark-400 tw:text-ink tw:enabled:hover:bg-mode-bark-500 tw:enabled:active:bg-mode-bark-600",
        danger: "tw:bg-danger tw:text-text-on-danger tw:enabled:hover:bg-danger-hover tw:enabled:active:bg-danger-pressed",
        // on the moss-600 nav bar. aria-expanded (its panel is open) holds the pressed fill.
        inverse:
          "tw:bg-transparent tw:text-text-inverse tw:enabled:hover:bg-moss-700 tw:enabled:active:bg-moss-800 tw:aria-expanded:bg-moss-800 tw:aria-pressed:bg-moss-800",
      },
      size: {
        sm: "tw:h-8 tw:px-3 tw:text-xs",
        md: "tw:h-10 tw:px-5 tw:text-sm",
        lg: "tw:h-12 tw:px-6 tw:text-base",
      },
      shape: {
        rounded: "tw:rounded-button",
        circle: "tw:rounded-full", // icon-only
      },
      // components pass this (default: sm for sm/md buttons, md for lg) so exactly one size class applies
      iconSize: { sm: "tw:[&_svg]:size-(--icon-sm)", md: "tw:[&_svg]:size-(--icon-md)" },
      iconStart: { true: "" },
      iconEnd: { true: "" },
      iconOnly: { true: "tw:px-0" },
    },
    compoundVariants: [
      // the icon side gets 4px less padding so the pill looks balanced
      { size: "sm", iconStart: true, className: "tw:ps-2" },
      { size: "md", iconStart: true, className: "tw:ps-4" },
      { size: "lg", iconStart: true, className: "tw:ps-5" },
      { size: "sm", iconEnd: true, className: "tw:pe-2" },
      { size: "md", iconEnd: true, className: "tw:pe-4" },
      { size: "lg", iconEnd: true, className: "tw:pe-5" },
      // icon-only buttons are square
      { size: "sm", iconOnly: true, className: "tw:w-8" },
      { size: "md", iconOnly: true, className: "tw:w-10" },
      { size: "lg", iconOnly: true, className: "tw:w-12" },
    ],
    defaultVariants: { variant: "primary", size: "md", shape: "rounded" },
  },
);

export type ButtonVariants = VariantProps<typeof button>;
export const defaultIconSize = (size?: "sm" | "md" | "lg" | null) => (size === "lg" ? "md" : "sm");
export const buttonVariants = ["primary", "secondary", "outline", "ghost", "accent", "danger", "inverse"] as const;
export const buttonSizes = ["sm", "md", "lg"] as const;
