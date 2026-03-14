'use client';

import { useEffect, useState } from 'react';
import { api, ensureValidToken } from '@/lib/api';
import { openTelegramLink } from '@/lib/telegram';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { LoadingScreen } from '@/components/LoadingScreen';

const BOT_USERNAME =
  typeof process !== 'undefined' ? process.env.NEXT_PUBLIC_BOT_USERNAME || 'YourBotUsername' : 'YourBotUsername';

export default function WaitlistPage() {
  const [channelCount, setChannelCount] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    ensureValidToken()
      .then((token) => {
        if (!token) {
          setLoading(false);
          return;
        }
        return api<{ channel_count: number }>('/api/v1/market/waitlist').then((res) => {
          if (res.ok && res.data) {
            setChannelCount(res.data.channel_count);
          }
        });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  return (
    <>
      <div className={`flex min-h-screen flex-col items-center justify-center px-4 ${loading ? 'opacity-0' : 'opacity-100'}`}>
        <PageTopSpacer />
        {channelCount !== null && channelCount === 0 && (
          <div className="flex flex-col items-center space-y-6 text-center">
            <h1 className="text-2xl font-bold">Waitlist is opened!</h1>
            <p className="text-muted-foreground">
              Connect your channel to try market first!
            </p>
            <button
              type="button"
              onClick={() => {
                const link = `https://t.me/${BOT_USERNAME}?startchannel&admin=post_messages+edit_messages+delete_messages+post_stories+delete_stories+manage_chat`;
                openTelegramLink(link);
              }}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-3 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              Add Channel
            </button>
          </div>
        )}
        {channelCount !== null && channelCount > 0 && (
          <div className="flex flex-col items-center space-y-4 text-center">
            <h1 className="text-2xl font-bold">You will be first in line! 🎉</h1>
          </div>
        )}
      </div>
      <LoadingScreen show={loading} />
    </>
  );
}
