'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api, auth, setAuthToken } from '@/lib/api';
import type { Listing } from '@/types';
import { ListingCard } from '@/components/ListingCard';
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

  return (
    <div className="min-h-screen pb-20">
      <div className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur">
        <div className="mx-auto max-w-4xl px-4 py-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <h1 className="text-2xl font-bold">My Listings</h1>
              <p className="text-sm text-muted-foreground">Manage your channel and advertiser listings</p>
            </div>
            <Link
              href="/listings/create"
              className="shrink-0 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90"
            >
              Create listing
            </Link>
          </div>
        </div>
      </div>
      <div className="mx-auto max-w-4xl px-4 py-4">
        {authed === false && (
          <p className="py-8 text-center text-muted-foreground">
            Open from Telegram to see your listings.
          </p>
        )}
        {authed && loading && (
          <div className="flex justify-center py-8">
            <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          </div>
        )}
        {authed && !loading && (
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
    </div>
  );
}
