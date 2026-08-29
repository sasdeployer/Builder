import { defineConfig } from "vite";
import { wgslVitePlugin } from "@vgpu/wgsl/loader-vite";
import { resolve } from "node:path";

export default defineConfig({
  plugins: [wgslVitePlugin()],
  build: {
    rollupOptions: {
      input: {
        main: resolve(__dirname, "index.html"),
        icon: resolve(__dirname, "icon.html"),
      },
    },
  },
});
