import { MoonIcon, SunIcon } from "@phosphor-icons/react";
import { useEffect, useLayoutEffect } from "react";
import { Alert, IconButton, Panel, Spinner } from "./ds";
import { readSingleWidget } from "./query";
import { useSetting } from "./settings";
import { SingleWidget } from "./SingleWidget";
import { usePageData, type PageData } from "./usePageData";
import { CpuWidget } from "./widgets/Cpu";
import { GpuWidget } from "./widgets/Gpu";
import { NetworkWidget } from "./widgets/Network";
import { OsWidget } from "./widgets/Os";
import { RamWidget } from "./widgets/Ram";
import { StorageWidget } from "./widgets/Storage";

const systemDark = window.matchMedia("(prefers-color-scheme: dark)").matches;

export function App() {
  const data = usePageData();
  const single = readSingleWidget();
  const [dark, setDark] = useSetting("darkMode", systemDark);

  useLayoutEffect(() => {
    document.documentElement.dataset.theme = single?.theme ?? (dark ? "dark" : "light");
  }, [dark, single?.theme]);

  const title = data.info?.config.page_title;
  useEffect(() => {
    if (title) document.title = title;
  }, [title]);

  if (single) return <SingleWidget data={data} query={single} />;
  return <Dashboard data={data} dark={dark} setDark={setDark} />;
}

function Dashboard({ data, dark, setDark }: { data: PageData; dark: boolean; setDark: (v: boolean) => void }) {
  const { info } = data;
  const config = info?.config;
  const host = config?.show_host ? (config.custom_host ?? window.location.hostname) : undefined;
  const version = info?.os.dash_version;

  return (
    <div className="tw:box-border tw:min-h-screen tw:bg-canvas tw:p-2 tw:sm:p-4">
      <Panel as="main" padding="lg" overflow="visible" className="tw:mx-auto tw:box-border tw:flex tw:max-w-screen-2xl tw:flex-col tw:gap-6">
        <header className="tw:flex tw:items-center tw:justify-between tw:gap-4">
          <div className="tw:flex tw:min-w-0 tw:items-baseline tw:gap-3">
            <h1 className="tw:m-0 tw:title-2xl" title={config?.show_dash_version === "icon_hover" && version ? `dash-comma ${version}` : undefined}>
              {config?.page_title ?? "d,"}
            </h1>
            {host && <span className="tw:truncate tw:text-base tw:text-text-muted">{host}</span>}
          </div>
          <IconButton label={dark ? "Use light mode" : "Use dark mode"} icon={dark ? <SunIcon /> : <MoonIcon />} variant="ghost" onClick={() => setDark(!dark)} />
        </header>

        {data.error && (
          <Alert tone="danger" title="Can't reach the server">
            The page keeps trying to reconnect. Check dash-comma is running.
          </Alert>
        )}

        {!info || !config ? (
          !data.error && (
            <div className="tw:flex tw:justify-center tw:py-16">
              <Spinner />
            </div>
          )
        ) : (
          <div className="tw:flex tw:flex-wrap tw:gap-4">
            {config.widget_list.map((w) => {
              switch (w) {
                case "os":
                  return <OsWidget key={w} os={info.os} config={config} />;
                case "cpu":
                  return <CpuWidget key={w} info={info.cpu} load={data.cpu} processes={data.cpuProcesses} config={config} />;
                case "ram":
                  return <RamWidget key={w} info={info.ram} load={data.ram} config={config} />;
                case "storage":
                  return <StorageWidget key={w} data={info.storage} load={data.storage} config={config} />;
                case "network":
                  return <NetworkWidget key={w} info={info.network} load={data.network} config={config} />;
                case "gpu":
                  return <GpuWidget key={w} info={info.gpu} load={data.gpu} config={config} />;
                default:
                  return null;
              }
            })}
          </div>
        )}

        {config?.show_dash_version === "bottom_right" && version && (
          <p className="tw:m-0 tw:self-end tw:text-xs tw:text-text-subtle">dash-comma {version}</p>
        )}
      </Panel>
    </div>
  );
}
