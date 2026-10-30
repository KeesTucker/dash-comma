import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// base "./" so the built UI works under any DASHDOT_ROUTING_PATH. `bun run
// dev` talks to a local dash-comma on :3001 (see usePageData).
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: "./",
  build: { outDir: "dist", emptyOutDir: true },
});
