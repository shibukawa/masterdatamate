import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { resolve } from "node:path";

export default defineConfig({
  root: "enemy-status-editor-react",
  base: "./",
  plugins: [react()],
  build: {
    outDir: "../enemy-status-editor",
    emptyOutDir: true,
    rollupOptions: {
      input: resolve("enemy-status-editor-react/index.html")
    }
  }
});
