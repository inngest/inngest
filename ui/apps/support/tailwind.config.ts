import baseConfig from "../../packages/components/tailwind.config";

/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
    "../../packages/components/src/**/*.{ts,tsx}",
  ],
  presets: [baseConfig],
};
