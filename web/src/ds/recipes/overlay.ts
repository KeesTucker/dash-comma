import { cva } from "class-variance-authority";

// ---------- Panel: the white container pages are built in
export const panel = cva(
  "tw:relative tw:box-border tw:flex tw:min-h-0 tw:min-w-0 tw:flex-col tw:rounded-xl tw:bg-surface-raised tw:font-sans tw:text-text tw:inset-ring tw:inset-ring-border",
  {
    variants: {
      padding: { none: "", md: "tw:p-6", lg: "tw:p-8" },
      // scroll: the panel scrolls its own content (pages); visible: content decides
      overflow: { scroll: "tw:overflow-y-auto", visible: "" },
    },
    defaultVariants: { padding: "md", overflow: "scroll" },
  },
);
