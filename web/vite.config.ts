import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  plugins: [react(), tailwindcss(), tsconfigPaths()],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8443",
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ["react", "react-dom", "react-router-dom"],
          query: ["@tanstack/react-query"],
          ui: ["lucide-react", "sonner"],
          codemirror: [
            "@codemirror/view",
            "@codemirror/state",
            "@codemirror/language",
            "@codemirror/commands",
            "@codemirror/search",
            "@codemirror/autocomplete",
            "@codemirror/lang-html",
            "@codemirror/lang-css",
            "@codemirror/lang-javascript",
            "@codemirror/lang-php",
            "@codemirror/lang-json",
            "@codemirror/lang-xml",
            "@codemirror/lang-markdown",
            "@codemirror/lang-python",
            "@codemirror/lang-sql",
            "@codemirror/lang-yaml",
            "@codemirror/theme-one-dark",
          ],
        },
      },
    },
  },
});
