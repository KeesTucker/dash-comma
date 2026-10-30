import { cva } from "class-variance-authority";

export type Crayon = "coral" | "orange" | "lime" | "green" | "teal" | "blue" | "purple" | "pink";

// ---------- crayon tiles: a crayon 100 square with the glyph in the crayon 700
// written out in full: Tailwind cannot see class names built at runtime
export const crayonTile: Record<Crayon, string> = {
  coral: "tw:bg-mode-crayon-coral-100 tw:text-mode-crayon-coral-700",
  orange: "tw:bg-mode-crayon-orange-100 tw:text-mode-crayon-orange-700",
  lime: "tw:bg-mode-crayon-lime-100 tw:text-mode-crayon-lime-700",
  green: "tw:bg-mode-crayon-green-100 tw:text-mode-crayon-green-700",
  teal: "tw:bg-mode-crayon-teal-100 tw:text-mode-crayon-teal-700",
  blue: "tw:bg-mode-crayon-blue-100 tw:text-mode-crayon-blue-700",
  purple: "tw:bg-mode-crayon-purple-100 tw:text-mode-crayon-purple-700",
  pink: "tw:bg-mode-crayon-pink-100 tw:text-mode-crayon-pink-700",
};

// sm 28, md 32, lg 40
export const iconTile = cva("tw:box-border tw:flex tw:shrink-0 tw:items-center tw:justify-center tw:rounded-lg", {
  variants: {
    size: {
      sm: "tw:size-7 tw:[&_svg]:size-(--icon-sm)",
      md: "tw:size-8 tw:[&_svg]:size-(--icon-md)",
      lg: "tw:size-10 tw:[&_svg]:size-(--icon-md)",
    },
  },
  defaultVariants: { size: "md" },
});

// ---------- SectionHeader: icon tile top left, title, actions on the right
export const sectionHeader = {
  root: "tw:box-border tw:flex tw:flex-wrap tw:items-center tw:gap-x-3 tw:gap-y-2 tw:font-sans",
  start: "tw:flex tw:min-w-0 tw:grow tw:items-center tw:gap-2",
  title: "tw:m-0 tw:min-w-0 tw:title-base tw:text-text",
  actions: "tw:flex tw:flex-wrap tw:items-center tw:gap-1",
};
