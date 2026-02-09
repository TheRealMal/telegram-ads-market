import Script from 'next/script';
import './globals.css';
import { BottomNav } from '@/components/BottomNav';

export const metadata = {
  title: 'ADS Marketplace',
  description: 'Telegram Mini App — ad space marketplace',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <Script
          src="https://telegram.org/js/telegram-web-app.js"
          strategy="beforeInteractive"
        />
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){try{var stored=typeof window!=='undefined'&&localStorage.getItem('ads_mrkt_theme');var s=stored==='light'||stored==='dark'?stored:(typeof window!=='undefined'&&window.Telegram&&window.Telegram.WebApp&&window.Telegram.WebApp.colorScheme)||'light';document.documentElement.classList.toggle('dark',s==='dark');}catch(e){}})();`,
          }}
        />
      </head>
      <body className="min-h-screen bg-background text-foreground antialiased">
        <main className="mx-auto max-w-4xl pb-20">{children}</main>
        <BottomNav />
      </body>
    </html>
  );
}
