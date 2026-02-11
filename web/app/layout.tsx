import Script from 'next/script';
import './globals.css';
import { BottomNav } from '@/components/BottomNav';
import { TonConnectProvider } from '@/components/TonConnectProvider';
import { TelegramExpandOnMobile } from '@/components/TelegramExpandOnMobile';
import { ThemeFromWebApp } from '@/components/ThemeFromWebApp';

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
            __html: `(function(){var k='ads_mrkt_theme';var s=typeof localStorage!=='undefined'?localStorage.getItem(k):null;if(s==='light'||s==='dark'){document.documentElement.classList.toggle('dark',s==='dark');return;}var c=typeof window!=='undefined'&&window.Telegram&&window.Telegram.WebApp&&window.Telegram.WebApp.colorScheme?window.Telegram.WebApp.colorScheme:'light';document.documentElement.classList.toggle('dark',c==='dark');})();`,
          }}
        />
      </head>
      <body className="min-h-screen bg-background text-foreground antialiased">
        <TonConnectProvider>
          <ThemeFromWebApp />
          <TelegramExpandOnMobile />
          <main className="mx-auto max-w-4xl pb-20">{children}</main>
          <BottomNav />
        </TonConnectProvider>
      </body>
    </html>
  );
}
