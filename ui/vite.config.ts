import react from "@vitejs/plugin-react";
import path from "path";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@gen": path.resolve(__dirname, "../gen/ts"),
      // @bufbuild/protobuf v2 uses subpath exports (/codegenv2, /wkt) that
      // can't be resolved by Vite when the importing file lives outside the
      // ui/ root (e.g. gen/ts/).  Redirect all @bufbuild/protobuf/* imports
      // to the local ESM dist tree so Vite finds them unconditionally.
      "@bufbuild/protobuf": path.resolve(
        __dirname,
        "node_modules/@bufbuild/protobuf/dist/esm",
      ),
    },
  },
  server: {
    proxy: {
      "/openmenses.v1.CycleTrackerService/": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
  },
});
