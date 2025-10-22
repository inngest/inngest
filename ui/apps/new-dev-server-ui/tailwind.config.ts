/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './src/**/*.{js,jsx,ts,tsx}',
    '../../packages/components/src/**/*.{ts,tsx}',
  ],
  presets: [require('../../packages/components/tailwind.config.ts')],
}
