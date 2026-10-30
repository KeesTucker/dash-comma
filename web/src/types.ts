// The API's shapes, unchanged from dashdot's @dashdot/common.

export type CpuInfo = { brand: string; model: string; cores: number; ecores: number; pcores: number; threads: number; frequency: number };
export type CpuLoad = { core: number; load: number; temp: number }[];

export type RamInfo = { size: number; layout: { brand?: string; type?: string; frequency?: number }[] };
export type RamLoad = number;

export type StorageInfo = {
  raidType?: 0 | 1;
  raidLabel?: string;
  raidName?: string;
  size: number;
  virtual?: boolean;
  disks: { device: string; brand: string; type: string }[];
}[];
export type StorageLoad = number[];

export type NetworkInfo = { interfaceSpeed: number; speedDown: number; speedUp: number; lastSpeedTest: number; type: string; publicIp: string };
export type NetworkLoad = { up: number; down: number };

export type GpuInfo = { layout: { brand: string; model: string; memory: number }[] };
export type GpuLoad = { layout: { load: number; memory: number; engines?: Record<string, number>; processes?: ProcessLoad[] }[] };

/** dash-comma's own: the busiest processes, on cpu-processes and in each GPU's load. */
export type ProcessLoad = { pid: number; name: string; load: number };

export type OsInfo = {
  platform: string;
  distro: string;
  release: string;
  kernel: string;
  arch: string;
  uptime: number;
  dash_version: string;
  dash_buildhash: string;
};

export type Widget = "os" | "cpu" | "storage" | "ram" | "network" | "gpu";

export type Config = {
  show_dash_version?: "icon_hover" | "bottom_right";
  show_host: boolean;
  custom_host?: string;
  page_title: string;
  use_imperial: boolean;
  network_speed_as_bytes: boolean;
  always_show_percentages: boolean;
  enable_cpu_temps: boolean;
  cpu_temps_mode: "max" | "avg";

  widget_list: Widget[];
  os_label_list: ("os" | "arch" | "up_since" | "dash_version")[];
  cpu_label_list: ("brand" | "model" | "cores" | "threads" | "frequency")[];
  storage_label_list: ("brand" | "size" | "type" | "raid")[];
  ram_label_list: ("brand" | "size" | "type" | "frequency")[];
  network_label_list: ("type" | "speed_up" | "speed_down" | "interface_speed" | "public_ip")[];
  gpu_label_list: ("brand" | "model" | "memory")[];

  os_widget_grow: number;
  os_widget_min_width: number;
  cpu_widget_grow: number;
  cpu_widget_min_width: number;
  cpu_shown_datapoints: number;
  cpu_cores_toggle_mode: "toggle" | "multi-core" | "average";
  storage_widget_items_per_page: number;
  storage_widget_grow: number;
  storage_widget_min_width: number;
  ram_widget_grow: number;
  ram_widget_min_width: number;
  ram_shown_datapoints: number;
  network_widget_grow: number;
  network_widget_min_width: number;
  network_shown_datapoints: number;
  gpu_widget_grow: number;
  gpu_widget_min_width: number;
  gpu_shown_datapoints: number;

  override: {
    os?: string;
    arch?: string;
    cpu_brand?: string;
    cpu_model?: string;
    cpu_cores?: number;
    cpu_threads?: number;
    cpu_frequency?: number;
    ram_brand?: string;
    ram_size?: number;
    ram_type?: string;
    ram_frequency?: number;
    network_type?: string;
    network_speed_up?: number;
    network_speed_down?: number;
    network_interface_speed?: number;
    network_public_ip?: string;
    storage_brands: Record<string, string>;
    storage_types: Record<string, string>;
    storage_sizes: Record<string, number>;
    gpu_brands: string[];
    gpu_models: string[];
    gpu_memories: number[];
  };
};

export type ServerInfo = {
  os: OsInfo;
  cpu: CpuInfo;
  ram: RamInfo;
  storage: StorageInfo;
  network: NetworkInfo;
  gpu: GpuInfo;
  config: Config;
};
