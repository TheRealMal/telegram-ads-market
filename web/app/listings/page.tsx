'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import type { Listing } from '@/types';
import { ListingCard } from '@/components/ListingCard';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { LoadingScreen } from '@/components/LoadingScreen';

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
    <>
      <div className={`min-h-screen pb-20 ${loading ? 'opacity-0' : 'opacity-100'}`}>
      <PageTopSpacer />
      <div className="mx-auto max-w-4xl px-4 py-4">
        {error && (
          <p className="py-8 text-center text-sm text-destructive">{error}</p>
        )}
        {!error && listings.length === 0 && (
          <p className="py-8 text-center text-muted-foreground">No listings yet.</p>
        )}
        {!error && listings.length > 0 && (
          <div className="grid gap-4">
            {listings.map((l) => (
              <ListingCard key={l.id} listing={l} />
            ))}
          </div>
        )}
      </div>
      </div>
      <LoadingScreen show={loading} />
    </>
  );
}
