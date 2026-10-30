import { DesktopIcon } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { duration } from "../format";
import type { Config, OsInfo } from "../types";
import { Widget } from "../Widget";
import { pick } from "./stats";

/** Uptime counts on in the browser between the server's static-info updates. */
function useUptime(uptime: number) {
  const [start] = useState(() => Date.now() / 1000 - uptime);
  const [now, setNow] = useState(() => Date.now() / 1000);
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now() / 1000), 30_000);
    return () => clearInterval(t);
  }, []);
  return now - start;
}

export function OsWidget({ os, config }: { os: OsInfo; config: Config }) {
  const uptime = useUptime(os.uptime);
  const o = config.override;
  const name = o.os ?? [os.distro, os.release !== "unknown" ? os.release : ""].filter(Boolean).join(" ");
  const stats = pick(config.os_label_list, {
    os: { label: "System", value: name },
    arch: { label: "Architecture", value: o.arch ?? os.arch },
    up_since: { label: "Up for", value: duration(uptime) },
    dash_version: { label: "dash-comma", value: os.dash_version },
  });
  return (
    <Widget title="Server" icon={DesktopIcon} hue="teal" grow={config.os_widget_grow} minWidth={config.os_widget_min_width} pages={[stats]}>
      <p className="tw:m-0 tw:text-sm tw:text-text-muted">Kernel {os.kernel}</p>
    </Widget>
  );
}
