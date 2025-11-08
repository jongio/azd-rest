import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import mdx from '@astrojs/mdx';
import expressiveCode from 'astro-expressive-code';

// https://astro.build/config
export default defineConfig({
  site: 'https://jongio.github.io/azd-rest/',
  base: '/azd-rest/',
  integrations: [
    expressiveCode(),
    mdx()
  ],
  vite: {
    plugins: [tailwindcss()]
  },
  output: 'static'
});
