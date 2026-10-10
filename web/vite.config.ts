import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/models": "http://localhost:8080",
      "/workflows": "http://localhost:8080",
      "/runs": "http://localhost:8080",
      "/artifacts": "http://localhost:8080",
      "/health": "http://localhost:8080",
    },
  },
});
