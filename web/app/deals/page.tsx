'use client';

import { useEffect, useState } from 'react';
import { api, ensureValidToken } from '@/lib/api';
import type { Deal } from '@/types';
import { DealCard } from '@/components/DealCard';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { LoadingScreen } from '@/components/LoadingScreen';

export default function DealsPage() {
  const [deals, setDeals] = useState<Deal[]>([]);
  const [loading, setLoading] = useState(true);
  const [authed, setAuthed] = useState<boolean | null>(null);

  useEffect(() => {
    ensureValidToken().then((token) => {
      if (token) {
        setAuthed(true);
      } else {
        setAuthed(false);
      }
    });
  }, []);

  useEffect(() => {
    if (!authed) {
      setLoading(false);
      return;
    }
    api<Deal[]>('/api/v1/market/my-deals')
      .then((res) => {
        if (res.ok && res.data) setDeals(res.data);
      })
      .finally(() => setLoading(false));
  }, [authed]);

  const ready = authed !== null && !loading;

  return (
    <>
      <div className={`page-with-nav ${ready ? 'opacity-100' : 'opacity-0'}`}>
      <PageTopSpacer />
      <div className="mx-auto max-w-4xl px-4 py-4">
        {authed === false && (
          <p className="py-8 text-center text-muted-foreground">
            Open from Telegram to see your deals.
          </p>
        )}
        {authed && deals.length === 0 && (
          <p className="py-8 text-center text-muted-foreground">No deals yet.</p>
        )}
        {authed && deals.length > 0 && (
          <div className="grid gap-4">
            {deals.map((d) => (
              <DealCard key={d.id} deal={d} />
            ))}
          </div>
        )}
      </div>
      </div>
      <LoadingScreen show={!ready} />
    </>
  );
}
