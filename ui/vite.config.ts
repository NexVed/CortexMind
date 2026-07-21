import { defineConfig } from 'vite';
import solidPlugin from 'vite-plugin-solid';

export default defineConfig({
  plugins: [solidPlugin()],
  server: {
    port: 3000,
    proxy: {
      // Local API + OAuth redirects
      '/api': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
      // Local API
      '/_': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
      // ConnectRPC services (cortex.v1.*)
      '/cortex.v1.': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'esnext',
  },
});
