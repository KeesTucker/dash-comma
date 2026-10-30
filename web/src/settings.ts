import { useCallback, useEffect, useState } from "react";

type Settings = { darkMode: boolean; multiCore: boolean; splitStorage: boolean };

// Same keys and JSON as dashdot's store.js, so saved choices carry over.
const read = <K extends keyof Settings>(key: K): Settings[K] | undefined => {
  try {
    const v = localStorage.getItem(key);
    return v == null ? undefined : (JSON.parse(v) as Settings[K]);
  } catch {
    return undefined;
  }
};

const listeners = new Set<() => void>();

export function useSetting<K extends keyof Settings>(key: K, fallback: Settings[K]): [Settings[K], (v: Settings[K]) => void] {
  const [value, setValue] = useState<Settings[K]>(() => read(key) ?? fallback);

  useEffect(() => {
    const sync = () => setValue(read(key) ?? fallback);
    listeners.add(sync);
    return () => {
      listeners.delete(sync);
    };
  }, [key, fallback]);

  const set = useCallback(
    (v: Settings[K]) => {
      localStorage.setItem(key, JSON.stringify(v));
      listeners.forEach((l) => l());
    },
    [key],
  );
  return [value, set];
}
