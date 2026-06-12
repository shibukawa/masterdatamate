import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";
import { resolve } from "node:path";

export default defineConfig({
  root: "maze-grid-editor-vue",
  base: "./",
  plugins: [vue()],
  build: {
    outDir: "../maze-grid-editor",
    emptyOutDir: true,
    rollupOptions: {
      input: resolve("maze-grid-editor-vue/index.html")
    }
  }
});
