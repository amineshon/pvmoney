import type { Metadata, Viewport } from "next";
import "./globals.css";
import { AppShell } from "@/components/AppShell";
import { LocaleProvider } from "@/lib/i18n";

export const metadata: Metadata = {
  title: "PVMoney",
  description: "Personal finance: accounts, assets, projects, debts",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  themeColor: "#07080c",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="fa" dir="rtl" suppressHydrationWarning>
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="" />
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Vazirmatn:wght@300;400;500;600;700;800&display=swap"
          rel="stylesheet"
        />
        <script
          dangerouslySetInnerHTML={{
            __html: `try{var l=localStorage.getItem('pvmoney.locale');if(l==='en'||l==='de'){document.documentElement.lang=l;document.documentElement.dir='ltr';document.documentElement.classList.add('ltr');}}catch(e){}`,
          }}
        />
      </head>
      <body className="font-sans antialiased">
        <div className="grain" />
        <LocaleProvider>
          <AppShell>{children}</AppShell>
        </LocaleProvider>
      </body>
    </html>
  );
}
