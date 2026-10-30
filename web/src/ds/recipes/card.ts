import { cva } from "class-variance-authority";

// a resting card: raised surface, hairline ring and the rest shadow
export const card = cva(
  "tw:relative tw:m-0 tw:box-border tw:flex tw:flex-col tw:overflow-hidden tw:rounded-xl tw:bg-surface-raised tw:font-sans tw:text-text tw:shadow-rest tw:inset-ring tw:inset-ring-border",
  {
    variants: {
      padding: { none: "", sm: "tw:gap-3 tw:p-4", md: "tw:gap-5 tw:p-6", lg: "tw:gap-6 tw:p-8" },
    },
    defaultVariants: { padding: "md" },
  },
);
