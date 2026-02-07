'use client';

import { useEffect, useState } from 'react';
import { api, auth, setAuthToken } from '@/lib/api';
import type { Listing, Deal } from '@/types';
import { DealCard } from '@/components/DealCard';

export default function DealsPage() {
  const [deals, setDeals] = useState<Deal[]>([]);
  const [loading, setLoading] = useState(true);
  const [authed, setAuthed] = useState<boolean | null>(null);

  useEffect(() => {
    auth().then((res) => {
      if (res.ok && res.data) {
        setAuthToken(res.data);
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
    api<Listing[]>('/api/v1/market/my-listings')
      .then((res) => {
        if (!res.ok || !res.data) return [];
        return Promise.all(
          res.data.map((l) => api<Deal[]>(`/api/v1/market/listings/${l.id}/deals`))
        );
      })
      .then((responses) => {
        const all: Deal[] = [];
        responses.forEach((r) => {
          if (r.ok && r.data) all.push(...r.data);
        });
        setDeals(all);
      })
      .finally(() => setLoading(false));
  }, [authed]);

  return (
    <div className="min-h-screen pb-20">
      <div className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur">
        <div className="mx-auto max-w-4xl px-4 py-4">
          <h1 className="text-2xl font-bold">Deals</h1>
          <p className="text-sm text-muted-foreground">Your active and past deals</p>
        </div>
      </div>
      <div className="mx-auto max-w-4xl px-4 py-4">
        {authed === false && (
          <p className="py-8 text-center text-muted-foreground">
            Open from Telegram to see your deals.
          </p>
        )}
        {authed && loading && (
          <div className="flex justify-center py-8">
            <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          </div>
        )}
        {authed && !loading && deals.length === 0 && (
          <p className="py-8 text-center text-muted-foreground">No deals yet.</p>
        )}
        {authed && !loading && deals.length > 0 && (
          <div className="grid gap-4">
            {deals.map((d) => (
              <DealCard key={d.id} deal={d} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
