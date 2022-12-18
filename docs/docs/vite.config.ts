//vite.config.ts

import { SearchPlugin } from "vitepress-plugin-search";
import { defineConfig } from "vite";

//default options
var options = {
  //  ...flexSearchIndexOptions,
  previewLength: 62,
  buttonLabel: "Search",
  placeholder: "Search...",
};

export default defineConfig({
  plugins: [SearchPlugin(options)],
});