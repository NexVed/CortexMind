import { defineConfig, loadEnv } from 'vite';
import solidPlugin from 'vite-plugin-solid';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', '');
  const target = env.CORTEX_API_TARGET || 'http://127.0.0.1:47831';
  return {
  plugins: [solidPlugin()],
  server: {
    port: 3000,
    proxy: {
      // Local API + OAuth redirects
      '/api': {
        target,
        changeOrigin: true,
      },
      '/mcp': {
        target,
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'esnext',
  },
  };
});
