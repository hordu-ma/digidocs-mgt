import { fileURLToPath, URL } from "node:url";
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({ resolvers: [ElementPlusResolver()] }),
    Components({ resolvers: [ElementPlusResolver()] }),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  build: {
    // element-plus is a large but intentionally isolated vendor chunk; raise the
    // warning threshold so the expected size doesn't show as a build warning.
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        // Split heavy, rarely-changing dependencies into their own chunks so
        // they stay cached across app deploys and don't bloat the entry chunk.
        manualChunks: {
          "vue-vendor": ["vue", "vue-router", "pinia"],
          "element-plus": ["element-plus", "@element-plus/icons-vue"],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:18081",
        changeOrigin: true,
      },
    },
  },
});
