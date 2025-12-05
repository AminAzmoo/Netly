import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: {
    dedupe: ['three', 'react-globe.gl'],
    alias: {
      three: 'three',
    },
  },
  optimizeDeps: {
    include: ['three', 'react-globe.gl'],
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          three: ['three'],
        },
      },
    },
  },
})
