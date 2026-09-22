import type { Metadata, Viewport } from "next";
import "./globals.css";
import { AppShell } from "@/components/AppShell";
import { LocaleProvider } from "@/lib/i18n";
import { gilroy, iranYekan } from "@/lib/fonts";

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
    <html lang="fa" dir="rtl" className={`${iranYekan.variable} ${gilroy.variable}`} suppressHydrationWarning>
      <head>
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
