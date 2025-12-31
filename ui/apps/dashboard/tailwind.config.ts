import baseConfig from '../../packages/components/tailwind.config';
//
// re-exporting baseConfig for upstream use by resolveConfig which does not traverse that
export { baseConfig };

/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './src/**/*.{js,jsx,ts,tsx}',
    '../../packages/components/src/**/*.{ts,tsx}',
  ],
  presets: [baseConfig],
};
