'use client';

import { useEffect, useState } from 'react';
import { api, auth, setAuthToken } from '@/lib/api';
import { getTelegramUser } from '@/lib/initData';
import type { Channel } from '@/types';
import { ChannelCard } from '@/components/ChannelCard';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { toggleTheme, getCurrentTheme } from '@/lib/theme';
import { User, HelpCircle, X, Sun, Moon } from 'lucide-react';
import { LoadingScreen } from '@/components/LoadingScreen';

const ADD_CHANNEL_USERNAME =
  typeof process !== 'undefined' ? process.env.NEXT_PUBLIC_ADD_CHANNEL_USERNAME || 'therealmal' : 'therealmal';

export default function ProfilePage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [hasToken, setHasToken] = useState<boolean | null>(null);
  const [tgUser, setTgUser] = useState<ReturnType<typeof getTelegramUser>>(null);
  const [showAddChannelModal, setShowAddChannelModal] = useState(false);
  const [theme, setTheme] = useState<'light' | 'dark'>(() =>
    typeof document !== 'undefined' && document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  );

  useEffect(() => {
    setTgUser(getTelegramUser());
    setTheme(getCurrentTheme());
  }, []);

  useEffect(() => {
    auth()
      .then((res) => {
        if (res.ok && res.data) {
          setAuthToken(res.data);
          setHasToken(true);
        } else {
          setHasToken(false);
        }
      })
      .catch(() => setHasToken(false));
  }, []);

  useEffect(() => {
    if (hasToken !== true) {
      setLoading(false);
      return;
    }
    let isMounted = true;
    const fetchChannels = () =>
      api<Channel[]>('/api/v1/market/my-channels')
        .then((res) => {
          if (isMounted && res.ok && res.data) setChannels(res.data);
        })
        .catch(() => {})
        .finally(() => {
          if (isMounted) setLoading(false);
        });
    fetchChannels();
    const interval = setInterval(fetchChannels, 3000);
    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [hasToken]);

  const displayName = tgUser
    ? [tgUser.first_name, tgUser.last_name].filter(Boolean).join(' ') || 'User'
    : 'Profile';
  const displayUsername = tgUser?.username ? `@${tgUser.username}` : null;
  const photoUrl = tgUser?.photo_url ?? null;
  const atUsername = ADD_CHANNEL_USERNAME.startsWith('@') ? ADD_CHANNEL_USERNAME : `@${ADD_CHANNEL_USERNAME}`;

  const notReady = loading || hasToken === null;

  return (
    <>
      <div className={`min-h-screen pb-20 ${notReady ? 'opacity-0' : 'opacity-100'}`}>
      <PageTopSpacer />
      <div className="bg-gradient-to-b from-primary/10 to-background pt-8 pb-4">
        <div className="mx-auto max-w-2xl px-4">
          <div className="flex flex-col items-center space-y-4 text-center">
            <div className="relative inline-block">
              <div className="flex h-20 w-20 items-center justify-center overflow-hidden rounded-full bg-primary/20">
                {photoUrl ? (
                  <img
                    src={photoUrl}
                    alt=""
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <User size={40} className="text-primary" />
                )}
              </div>
              <button
                type="button"
                onClick={() => setTheme(toggleTheme())}
                className="absolute -right-1 -top-1 flex h-8 w-8 items-center justify-center rounded-full border border-border bg-background shadow-sm hover:bg-accent hover:text-accent-foreground"
                aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
              >
                {theme === 'dark' ? (
                  <Sun size={16} className="text-foreground" />
                ) : (
                  <Moon size={16} className="text-foreground" />
                )}
              </button>
            </div>
            <div>
              <h1 className="text-2xl font-bold">{displayName}</h1>
              {displayUsername && (
                <p className="text-muted-foreground">{displayUsername}</p>
              )}
              {!hasToken && (
                <p className="mt-1 text-sm text-muted-foreground">
                  Open from Telegram to sign in
                </p>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Channels only — no tab bar */}
      <div className="mx-auto max-w-2xl px-4 py-5">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
            My channels
          </h2>
          <button
            type="button"
            onClick={() => setShowAddChannelModal(true)}
            className="inline-flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          >
            <HelpCircle size={16} />
            How to add a channel
          </button>
        </div>

        {hasToken === false && (
          <p className="py-8 text-center text-muted-foreground">
            Open this app from Telegram to see your profile.
          </p>
        )}
        {hasToken && (
          <div className="space-y-3">
            {channels.length === 0 ? (
              <p className="py-4 text-center text-sm text-muted-foreground">
                No channels linked. Use “How to add a channel” above.
              </p>
            ) : (
              channels.map((c) => <ChannelCard key={c.id} channel={c} />)
            )}
          </div>
        )}
      </div>

      {/* How to add channel — popup */}
      {showAddChannelModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={() => setShowAddChannelModal(false)}
          role="dialog"
          aria-modal="true"
          aria-labelledby="add-channel-title"
        >
          <div
            className="max-h-[85vh] w-full max-w-md overflow-auto rounded-xl border border-border bg-card p-5 shadow-lg"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <h3 id="add-channel-title" className="text-lg font-semibold">
                How to add a channel
              </h3>
              <button
                type="button"
                onClick={() => setShowAddChannelModal(false)}
                className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
                aria-label="Close"
              >
                <X size={20} />
              </button>
            </div>
            <div className="mt-4 space-y-3 text-sm text-muted-foreground">
              <p>To link your channel to the marketplace:</p>
              <ol className="list-inside list-decimal space-y-2">
                <li>Open your Telegram channel.</li>
                <li>
                  Add <strong className="text-foreground cursor-pointer" onClick={() => window.open(`https://t.me/${atUsername}`, '_blank')}>{atUsername}</strong> as a member.
                </li>
                <li>
                  Grant <strong className="text-foreground cursor-pointer" onClick={() => window.open(`https://t.me/${atUsername}`, '_blank')}>{atUsername}</strong> admin rights with:
                </li>
              </ol>
              <ul className="list-inside list-disc space-y-1 pl-2">
                <li>Post / Edit / Delete messages</li>
                <li>Post / Delete stories</li>
                <li>Statistics automatically fetched if available</li>
              </ul>
              <p className="pt-2">
                After that, your channel will appear here and you can create listings for it.
              </p>
            </div>
          </div>
        </div>
      )}
      </div>
      <LoadingScreen show={notReady} />
    </>
  );
}
