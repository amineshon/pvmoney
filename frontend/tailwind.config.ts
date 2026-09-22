import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-app)"],
        display: ["var(--font-app)"],
      },
      colors: {
        ink: {
          950: "#07080c",
          900: "#0c0e14",
          800: "#12151e",
          700: "#1a1f2b",
        },
        gold: {
          50: "#fbf6e8",
          200: "#ead9a0",
          400: "#e0c36a",
          500: "#c9a227",
          700: "#8a6d12",
        },
      },
      boxShadow: {
        glow: "0 0 40px rgba(224, 195, 106, 0.18)",
        card: "0 20px 50px rgba(0,0,0,0.35)",
      },
      backgroundImage: {
        mesh: "radial-gradient(1200px 600px at 100% -10%, rgba(201,162,39,0.16), transparent 50%), radial-gradient(900px 500px at -10% 110%, rgba(62,224,162,0.08), transparent 45%)",
      },
    },
  },
  plugins: [],
};

export default config;
