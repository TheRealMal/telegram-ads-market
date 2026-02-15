'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Plus } from 'lucide-react';
import { api, auth, setAuthToken } from '@/lib/api';
import type { Listing } from '@/types';
import { ListingCard } from '@/components/ListingCard';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { LoadingScreen } from '@/components/LoadingScreen';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

export default function MyListingsPage() {
  const [listings, setListings] = useState<Listing[]>([]);
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
        if (res.ok && res.data) setListings(res.data);
      })
      .finally(() => setLoading(false));
  }, [authed]);

  const active = listings.filter((l) => l.status === 'active');
  const inactive = listings.filter((l) => l.status === 'inactive');

  const ready = authed !== null && !loading;
  if (!ready) return <LoadingScreen />;

  return (
    <div className="min-h-screen pb-20">
      <PageTopSpacer />
      <div className="mx-auto max-w-4xl px-4 py-4">
        {authed === false && (
          <p className="py-8 text-center text-muted-foreground">
            Open from Telegram to see your listings.
          </p>
        )}
        {authed && (
          <Tabs defaultValue="active" className="w-full">
            <TabsList className="mb-4 grid w-full grid-cols-2">
              <TabsTrigger value="active">Active</TabsTrigger>
              <TabsTrigger value="inactive">Inactive</TabsTrigger>
            </TabsList>
            <TabsContent value="active" className="space-y-4">
              {active.length === 0 ? (
                <p className="py-8 text-center text-muted-foreground">No active listings.</p>
              ) : (
                active.map((l) => <ListingCard key={l.id} listing={l} />)
              )}
            </TabsContent>
            <TabsContent value="inactive" className="space-y-4">
              {inactive.length === 0 ? (
                <p className="py-8 text-center text-muted-foreground">No inactive listings.</p>
              ) : (
                inactive.map((l) => <ListingCard key={l.id} listing={l} />)
              )}
            </TabsContent>
          </Tabs>
        )}
      </div>
      {/* FAB: create listing, above tab bar */}
      <Link
        href="/listings/create"
        className="fixed bottom-24 right-4 z-40 flex h-14 w-14 items-center justify-center rounded-full border border-border bg-background text-muted-foreground shadow-md transition-colors hover:bg-muted hover:text-foreground"
        aria-label="Create listing"
      >
        <Plus size={24} strokeWidth={2} />
      </Link>
    </div>
  );
}
