/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: "class",
  content: [
    "./pages/**/*.{js,ts,jsx,tsx}",
    "./components/**/*.{js,ts,jsx,tsx}",
    "./shared/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        slate: {
          950: "#0C1323",
          1000: "#080D19",
        },
      },
      fontSize: {
        "5xl": ["3rem", "1.3"],
        "2xs": "0.625rem",
      },
      maxWidth: {
        "container-desktop": "1600px",
      },
    },
    fontFamily: {
      sans: ["Inter", "sans-serif"],
      sans: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol"',
    },
  },
  plugins: [require("@tailwindcss/typography")],
};
