import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  base: '/warehouse/',
  optimizeDeps: {
    exclude: ['lucide-react'],
  },
  build: {
    outDir: '../static',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/realms': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/assets': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
});