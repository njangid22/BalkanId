/* eslint-disable */
// @ts-nocheck
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        // Use CSS variables with alpha support for Tailwind's `/opacity` utilities
        'brand-primary': 'rgb(var(--brand-primary) / <alpha-value>)',
        'brand-accent': 'rgb(var(--brand-accent) / <alpha-value>)',
        'brand-surface': 'rgb(var(--brand-surface) / <alpha-value>)',
        'brand-muted': 'rgb(var(--brand-muted) / <alpha-value>)'
      }
    }
  },
  plugins: []
};
