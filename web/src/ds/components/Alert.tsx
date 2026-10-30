import { CaretDownIcon, CheckCircleIcon, InfoIcon, WarningCircleIcon, WarningIcon, XIcon } from "@phosphor-icons/react";
import { AnimatePresence, MotionConfig, motion } from "motion/react";
import { useId, useState, type ReactNode } from "react";
import { spring, transition } from "../motion";
import { button } from "../recipes/button";
import { alert, alertAction, alertParts, alertTitle, type Tone } from "../recipes/feedback";
import { IconButton } from "./IconButton";

const icons = { info: InfoIcon, success: CheckCircleIcon, warning: WarningIcon, danger: WarningCircleIcon };

export type AlertAction = { label: string; onClick: () => void; disabled?: boolean };

export type AlertProps = {
  tone?: Tone;
  /** One line: what happened or what is true. */
  title: ReactNode;
  /** What it means or what to do. */
  children?: ReactNode;
  /** The main thing to do, as a small filled button in the alert's colour. */
  primaryAction?: AlertAction;
  /** A second option, as a small outline button in the alert's colour. */
  secondaryAction?: AlertAction;
  /** Anything else (custom buttons or links), after the primary and secondary actions. */
  actions?: ReactNode;
  /** Shows a close button; the alert hides itself. Leave off for alerts that must stay until the problem is fixed. */
  dismissible?: boolean;
  /** Called when closed (also shows the close button on its own). */
  onDismiss?: () => void;
  /** Shows a caret that folds the alert down to its title. */
  collapsible?: boolean;
  /** Start folded (uncontrolled). */
  defaultCollapsed?: boolean;
  /** Folded (controlled), with onCollapsedChange. */
  collapsed?: boolean;
  onCollapsedChange?: (collapsed: boolean) => void;
  /** Announce it when it appears (after a save or a failed submit). Static page alerts leave this off. */
  announce?: boolean;
  className?: string;
};

/** A status message that stays on the page. */
export function Alert({
  tone = "info",
  title,
  children,
  primaryAction,
  secondaryAction,
  actions,
  dismissible = false,
  onDismiss,
  collapsible = false,
  defaultCollapsed = false,
  collapsed,
  onCollapsedChange,
  announce = false,
  className = "",
}: AlertProps) {
  const I = icons[tone];
  const bodyId = useId();
  const [closed, setClosed] = useState(false);
  const [ownCollapsed, setOwnCollapsed] = useState(defaultCollapsed);
  const folded = collapsible && (collapsed ?? ownCollapsed);
  const role = announce ? (tone === "danger" || tone === "warning" ? "alert" : "status") : undefined;
  const hasActions = Boolean(primaryAction || secondaryAction || actions);
  const hasMore = Boolean(children || hasActions);
  const actionButton = (a: AlertAction, kind: "primary" | "secondary") => (
    <button
      type="button"
      onClick={a.onClick}
      disabled={a.disabled}
      className={button({ variant: null, size: "sm", iconSize: "sm", className: alertAction({ kind, tone }) })}
    >
      {a.label}
    </button>
  );

  const toggle = () => {
    const next = !folded;
    if (collapsed === undefined) setOwnCollapsed(next);
    onCollapsedChange?.(next);
  };

  return (
    <MotionConfig reducedMotion="user">
      <AnimatePresence initial={false}>
        {!closed && (
          <motion.div
            key="alert"
            role={role}
            className={`${alert({ tone })} ${className}`}
            exit={{ opacity: 0, transition: transition.dash }}
          >
            <I weight="fill" aria-hidden />
            <div className={alertParts.body}>
              <p className={alertTitle({ tone })}>{title}</p>
              <AnimatePresence initial={false}>
                {hasMore && !folded && (
                  <motion.div
                    key="more"
                    id={bodyId}
                    className="tw:flex tw:flex-col tw:gap-1 tw:overflow-hidden"
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: "auto", opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    transition={spring.glide}
                  >
                    {children && <div className={alertParts.text}>{children}</div>}
                    {hasActions && (
                      <div className={alertParts.actions}>
                        {primaryAction && actionButton(primaryAction, "primary")}
                        {secondaryAction && actionButton(secondaryAction, "secondary")}
                        {actions}
                      </div>
                    )}
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
            {(collapsible || dismissible || onDismiss) && (
              <div className={alertParts.controls}>
                {collapsible && hasMore && (
                  <IconButton
                    label={folded ? "Show more" : "Show less"}
                    icon={<CaretDownIcon className={`tw:transition-transform tw:duration-(--duration-dash) tw:ease-standard ${folded ? "" : "tw:rotate-180"}`} />}
                    variant="ghost"
                    shape="circle"
                    size="sm"
                    aria-expanded={!folded}
                    aria-controls={bodyId}
                    onClick={toggle}
                  />
                )}
                {(dismissible || onDismiss) && (
                  <IconButton
                    label="Dismiss"
                    icon={<XIcon />}
                    variant="ghost"
                    shape="circle"
                    size="sm"
                    onClick={() => {
                      setClosed(true);
                      onDismiss?.();
                    }}
                  />
                )}
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </MotionConfig>
  );
}
