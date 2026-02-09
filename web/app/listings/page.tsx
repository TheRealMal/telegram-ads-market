'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import type { Listing } from '@/types';
import { ListingCard } from '@/components/ListingCard';
import { PageTopSpacer } from '@/components/PageTopSpacer';

export default function ListingsPage() {
  const [listings, setListings] = useState<Listing[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api<Listing[]>('/api/v1/market/listings')
      .then((res) => {
        if (res.ok && res.data) setListings(res.data);
        else setError(res.error_code || 'Failed to load');
      })
      .catch(() => setError('Network error'))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="min-h-screen pb-20">
      <PageTopSpacer />
      <div className="mx-auto max-w-4xl px-4 py-4">
        {loading && (
          <div className="flex justify-center py-8">
            <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          </div>
        )}
        {error && (
          <p className="py-8 text-center text-sm text-destructive">{error}</p>
        )}
        {!loading && !error && listings.length === 0 && (
          <p className="py-8 text-center text-muted-foreground">No listings yet.</p>
        )}
        {!loading && !error && listings.length > 0 && (
          <div className="grid gap-4">
            {listings.map((l) => (
              <ListingCard key={l.id} listing={l} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
