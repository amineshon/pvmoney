import localFont from "next/font/local";

export const iranYekan = localFont({
  src: [
    { path: "../public/font/IRANYekanLight.ttf", weight: "300", style: "normal" },
    { path: "../public/font/IRANYekanLight.ttf", weight: "400", style: "normal" },
    { path: "../public/font/IRANYekanBold.ttf", weight: "500", style: "normal" },
    { path: "../public/font/IRANYekanBold.ttf", weight: "600", style: "normal" },
    { path: "../public/font/IRANYekanBold.ttf", weight: "700", style: "normal" },
    { path: "../public/font/IRANYekanBlack.ttf", weight: "800", style: "normal" },
    { path: "../public/font/IRANYekanBlack.ttf", weight: "900", style: "normal" },
  ],
  variable: "--font-iran-yekan",
  display: "swap",
  fallback: ["Tahoma", "Arial", "sans-serif"],
});

export const gilroy = localFont({
  src: [
    { path: "../public/font/Gilroy-Light.ttf", weight: "300", style: "normal" },
    { path: "../public/font/Gilroy-Regular.ttf", weight: "400", style: "normal" },
    { path: "../public/font/Gilroy-Medium.ttf", weight: "500", style: "normal" },
    { path: "../public/font/Gilroy-Semibold.ttf", weight: "600", style: "normal" },
    { path: "../public/font/Gilroy-Bold.ttf", weight: "700", style: "normal" },
    { path: "../public/font/Gilroy-Extrabold.ttf", weight: "800", style: "normal" },
    { path: "../public/font/Gilroy-Black.ttf", weight: "900", style: "normal" },
  ],
  variable: "--font-gilroy",
  display: "swap",
  fallback: ["system-ui", "sans-serif"],
});
