import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/v1": "http://localhost:8080",
      // Only proxy the backend OAuth routes, not /auth/callback which is a frontend page.
      "/auth/google": "http://localhost:8080",
    },
  },
});
