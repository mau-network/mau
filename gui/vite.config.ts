import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  root: '.',
  publicDir: 'public',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: true,
  },
  server: {
    port: 3000,
  },
  resolve: {
    alias: {
      '@': '/src',
    },
    conditions: ['browser', 'import', 'module', 'default'],
  },
  optimizeDeps: {
    exclude: ['node-fetch', 'fetch-blob'],
  },
});
