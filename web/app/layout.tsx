import Script from 'next/script';
import './globals.css';
import { BottomNav } from '@/components/BottomNav';
import { TonConnectProvider } from '@/components/TonConnectProvider';
import { TelegramExpandOnMobile } from '@/components/TelegramExpandOnMobile';

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
      </head>
      <body className="min-h-screen bg-background text-foreground antialiased">
        <TonConnectProvider>
          <TelegramExpandOnMobile />
          <main className="mx-auto max-w-4xl pb-20">{children}</main>
          <BottomNav />
        </TonConnectProvider>
      </body>
    </html>
  );
}
