import react from "@vitejs/plugin-react-swc";
import { promises as fs } from "fs";
import { resolve } from "path";
import { defineConfig } from "vite";
// import mkcert from "vite-plugin-mkcert";
import { VitePWA } from "vite-plugin-pwa";
import { viteStaticCopy } from "vite-plugin-static-copy";

const backend = "http://localhost:5212";
const backendProxyOption = { target: backend, headers: { host: new URL(backend).host } };

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "prompt",
      injectRegister: "auto",
      manifest: false,
      workbox: {
        globIgnores: ["**/*leaflet*", "**/*mapbox*", "**/*Leaflet*", "**/*Mapbox*"],
        maximumFileSizeToCacheInBytes: 10000000,
        navigateFallbackDenylist: [/^\/pdfviewer.html/, /^\/api\/(.+)/, /^\/f\/(.+)/, /^\/s\/(.+)/],
      },
      devOptions: {
        enabled: true,
      },
    }),
    viteStaticCopy({
      targets: [
        {
          src: "node_modules/pdfjs-dist/build/*.mjs",
          dest: "assets/pdfjs",
        },
      ],
    }),
    {
      name: "load-stylesheet-async",
      transformIndexHtml(html) {
        return html.replace(
          /<link rel="stylesheet" crossorigin href="(.+?)">/g,
          `<link rel="stylesheet" crossorigin href="$1" media="print" onload="this.media='all'">`,
        );
      },
    },
    {
      name: "generate-version",
      async writeBundle(outputOptions) {
        const version = {
          name: process.env.npm_package_name,
          version: process.env.npm_package_version,
        };
        const path = resolve(__dirname, outputOptions.dir, "version.json");
        await fs.writeFile(path, JSON.stringify(version));
      },
    },
    // mkcert({
    //   hosts: ["devv5.cloudreve.org"],
    // }),
  ],
  define: {
    __ASSETS_VERSION__: JSON.stringify(process.env.npm_package_version),
  },
  build: {
    outDir: "build", // keep same as v3 with minimal changes
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          const chunkMap = {
            common: [
              "vite/preload-helper",
              "vite/modulepreload-polyfill",
              "vite/dynamic-import-helper",
              "commonjsHelpers",
              "commonjs-dynamic-modules",
              "__vite-browser-external",
            ],
            monaco: ["monaco-editor"],
            codemirror: ["@codemirror"],
            excalidraw: [
              "node_modules/@excalidraw",
              "node_modules/browser-fs-access",
              "node_modules/image-blob-reduce",
              "node_modules/pica/",
            ],
            mermaid: ["node_modules/mermaid", "node_modules/katex"],
            leaflet: ["node_modules/leaflet", "node_modules/react-leaflet"],
            react: ["node_modules/react", "node_modules/react-dom"],
            mapbox: ["node_modules/mapbox-gl"],
          };

          // https://github.com/vitejs/vite/issues/5189#issuecomment-2175410148
          for (const [chunkName, patterns] of Object.entries(chunkMap)) {
            if (patterns.some((pattern) => id.includes(pattern))) {
              return chunkName;
            }
          }
        },
      },
    },
  },
  server: {
    host: "0.0.0.0",
    cors: false,
    proxy: {
      "/api": backendProxyOption,
      "/s/": backendProxyOption,
      "/f/": backendProxyOption,
      "/manifest.json": backendProxyOption,
    },
  },
});
