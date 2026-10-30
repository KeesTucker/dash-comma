// Motion (motion/react) transitions, matching the --duration-* and --ease-* tokens CSS uses.
export const ease = {
  standard: [0.2, 0, 0, 1] as [number, number, number, number],
  out: [0, 0, 0.2, 1] as [number, number, number, number],
  in: [0.4, 0, 1, 1] as [number, number, number, number],
};
/** Durations with the standard easing, e.g. transition={transition.dash}. */
export const transition = {
  zap: { duration: 0.1, ease: ease.standard },
  dash: { duration: 0.15, ease: ease.standard },
  leap: { duration: 0.2, ease: ease.standard },
  soar: { duration: 0.3, ease: ease.standard },
};
/** Springs, e.g. transition={spring.glide}. */
export const spring = {
  bounce: { type: "spring" as const, stiffness: 600, damping: 30, mass: 1 },
  glide: { type: "spring" as const, stiffness: 400, damping: 38, mass: 1 },
};
