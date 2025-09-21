// @ts-nocheck
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        "brand-primary": "#0B1F4C",
        "brand-accent": "#1B5CFF",
        "brand-surface": "#0F1729",
        "brand-muted": "#1E293B"
      },
      backgroundImage: {
        "gradient-hero": "linear-gradient(135deg, #020617 0%, #0B1F4C 50%, #2563EB 100%)",
        "gradient-card": "linear-gradient(180deg, rgba(30,64,175,0.35) 0%, rgba(2,6,23,0.9) 100%)"
      },
      boxShadow: {
        glow: "0 20px 45px -15px rgba(37, 99, 235, 0.45)",
        surface: "0 15px 35px -15px rgba(15, 23, 42, 0.5)"
      }
    }
  },
  plugins: []
};
