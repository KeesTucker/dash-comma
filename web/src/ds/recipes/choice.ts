// The Switch. Named group (group/choice) so it never reads an ancestor `group`'s state.
// The native input sits invisibly over the track (target-min square), so keyboard, forms and
// screen readers work unchanged; the track reads its state through `peer-*` and `group-has-*`.
import { cva } from "class-variance-authority";

export const choice = {
  root: cva("tw:group/choice tw:relative tw:inline-flex tw:items-start tw:gap-3 tw:font-sans tw:text-sm tw:text-text", {
    variants: {
      disabled: { true: "tw:opacity-(--opacity-disabled) tw:cursor-not-allowed", false: "tw:cursor-pointer" },
    },
    defaultVariants: { disabled: false },
  }),
  // one text line tall, so the box centres on the first line of the label
  slot: "tw:relative tw:flex tw:h-(--leading-sm) tw:shrink-0 tw:items-center tw:justify-center",
  input:
    "tw:peer tw:absolute tw:top-1/2 tw:left-1/2 tw:m-0 tw:size-(--target-min) tw:-translate-1/2 tw:cursor-pointer tw:disabled:cursor-not-allowed tw:appearance-none tw:opacity-0",
  text: "tw:flex tw:flex-col tw:gap-0.5",
  label: "tw:text-sm tw:text-text",
  description: "tw:text-sm tw:text-text-muted",
};

const motion = "tw:transition-[background-color,box-shadow] tw:duration-(--duration-dash) tw:ease-standard";
const focus = "tw:peer-focus-visible:shadow-focus";

export const switchTrack = cva(
  [
    "tw:pointer-events-none tw:box-border tw:flex tw:h-6 tw:w-10 tw:shrink-0 tw:items-center tw:rounded-full tw:p-0.5",
    "tw:bg-border-control tw:peer-checked:bg-primary",
    motion,
    focus,
  ],
  {
    variants: {
      disabled: { true: "", false: "tw:group-hover/choice:bg-border-control-hover tw:group-hover/choice:peer-checked:bg-primary-hover" },
    },
    defaultVariants: { disabled: false },
  },
);

export const switchThumb =
  "tw:size-5 tw:rounded-full tw:bg-knob tw:shadow-rest tw:transition-transform tw:duration-(--duration-dash) tw:ease-standard tw:group-has-checked/choice:translate-x-4";
