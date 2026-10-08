import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'node:path'

export default defineConfig({
  root: resolve(__dirname, 'preview'),
  base: './',
  publicDir: false,
  plugins: [vue()],
  resolve: { alias: { '@': resolve(__dirname, 'src'), 'vue-i18n': 'vue-i18n/dist/vue-i18n.runtime.esm-bundler.js' } },
  define: { __INTLIFY_JIT_COMPILATION__: true },
  build: { outDir: resolve(__dirname, 'preview-dist'), emptyOutDir: true },
})
