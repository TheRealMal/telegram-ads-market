'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { MessageCircle, BarChart3, Power, PowerOff, Trash2, X } from 'lucide-react';
import { api, auth, setAuthToken } from '@/lib/api';
import { useTelegramBackButton } from '@/lib/telegram';
import { parseListingPrices, formatPriceEntry, formatPriceKey, formatPriceValue } from '@/lib/formatPrice';
import type { Listing, Deal, Channel } from '@/types';
import { getDealStatusLabel } from '@/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { LoadingScreen } from '@/components/LoadingScreen';

function formatFollowers(n: number): string {
  if (n >= 1e6) return `${(n / 1e6).toFixed(1).replace(/\.0$/, '')}M`;
  if (n >= 1e3) return `${(n / 1e3).toFixed(1).replace(/\.0$/, '')}k`;
  return n.toLocaleString();
}

export default function ListingDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = Number(params?.id);
  useTelegramBackButton(() => router.back());
  const [listing, setListing] = useState<Listing | null>(null);
  const [deals, setDeals] = useState<Deal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isOwner, setIsOwner] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [showCreateDealModal, setShowCreateDealModal] = useState(false);
  const [dealMessage, setDealMessage] = useState('');
  const [dealPostedAt, setDealPostedAt] = useState('');
  const [createDealPriceIndex, setCreateDealPriceIndex] = useState(0);
  const [createDealChannelId, setCreateDealChannelId] = useState<number | null>(null);
  const [myChannels, setMyChannels] = useState<Channel[]>([]);
  const [createDealSubmitting, setCreateDealSubmitting] = useState(false);

  useEffect(() => {
    if (!id || Number.isNaN(id)) {
      setError('Invalid listing ID');
      setLoading(false);
      return;
    }
    api<Listing>(`/api/v1/market/listings/${id}`)
      .then((listRes) => {
        if (listRes.ok && listRes.data) setListing(listRes.data);
        else setError(listRes.error_code || 'Not found');
      })
      .catch(() => setError('Network error'))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (!listing || !id) return;
    auth().then((res) => {
      if (!res.ok || !res.data) return;
      setAuthToken(res.data);
      api<Listing[]>('/api/v1/market/my-listings').then((myRes) => {
        if (myRes.ok && myRes.data) setIsOwner(myRes.data.some((l) => l.id === id));
      });
      api<Deal[]>(`/api/v1/market/listings/${id}/deals`).then((dealsRes) => {
        if (dealsRes.ok && dealsRes.data) setDeals(dealsRes.data);
      });
    });
  }, [listing, id]);

  const priceRowsForListing = parseListingPrices(listing?.prices ?? null);
  const selectedPriceRow = priceRowsForListing[createDealPriceIndex];

  const handleOpenCreateDeal = async () => {
    setDealMessage('');
    setDealPostedAt('');
    setCreateDealPriceIndex(0);
    setCreateDealChannelId(null);
    setShowCreateDealModal(true);
    if (listing?.type === 'lessee') {
      const authRes = await auth();
      if (authRes.ok && authRes.data) {
        setAuthToken(authRes.data);
        const res = await api<Channel[]>('/api/v1/market/my-channels');
        if (res.ok && res.data) setMyChannels(res.data);
        else setMyChannels([]);
      }
    } else {
      setMyChannels([]);
    }
  };

  const handleCreateDeal = async () => {
    const authRes = await auth();
    if (!authRes.ok || !authRes.data) {
      alert('Please open the app from Telegram to create a deal.');
      return;
    }
    setAuthToken(authRes.data);
    const row = selectedPriceRow ?? priceRowsForListing[0];
    if (!row || !listing) {
      alert('No price configured for this listing.');
      return;
    }
    const type = row.duration + 'hr';
    const duration = parseInt(row.duration, 10) || 24;
    setCreateDealSubmitting(true);
    const details: { message?: string; posted_at?: string } = {
      message: dealMessage.trim() || undefined,
    };
    if (dealPostedAt.trim()) {
      try {
        const d = new Date(dealPostedAt.trim());
        if (!Number.isNaN(d.getTime())) details.posted_at = d.toISOString();
      } catch {
        /* ignore invalid date */
      }
    }
    if (listing.type === 'lessee' && !createDealChannelId) {
      alert('Please select a channel for your ad.');
      setCreateDealSubmitting(false);
      return;
    }
    const body: { listing_id: number; channel_id?: number; type: string; duration: number; price: number; details: object } = {
      listing_id: id,
      type,
      duration,
      price: row.price,
      details,
    };
    if (listing.type === 'lessee' && createDealChannelId != null) {
      body.channel_id = createDealChannelId;
    }
    const res = await api<Deal>('/api/v1/market/deals', {
      method: 'POST',
      body: JSON.stringify(body),
    });
    setCreateDealSubmitting(false);
    if (res.ok && res.data) {
      setShowCreateDealModal(false);
      router.push(`/deals/${res.data.id}`);
    } else {
      alert(res.error_code || 'Failed to create deal');
    }
  };

  const content = ( loading ? (
    <div className="min-h-screen" aria-hidden />
  ) : (error || !listing) ? (
    <div className="page-with-nav">
      <PageTopSpacer />
      <div className="mx-auto max-w-3xl px-4 py-8">
        <p className="text-destructive">{error || 'Not found'}</p>
      </div>
    </div>
  ) : (
    <div className="page-with-nav">
      <PageTopSpacer />
      <div className="mx-auto max-w-3xl space-y-6 px-4 py-4">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <Badge variant={listing.type === 'lessor' ? 'default' : 'secondary'}>
                {listing.type === 'lessor' ? 'Channel Owner' : 'Advertiser Request'}
              </Badge>
              <Badge variant={listing.status === 'active' ? 'outline' : 'secondary'}>
                {listing.status}
              </Badge>
            </div>
            {(listing.channel_title != null || listing.channel_username != null) && (
              <p className="mt-2 text-sm text-muted-foreground">
                Channel:{' '}
                {listing.channel_username ? (
                  <a
                    href={`https://t.me/${listing.channel_username}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary underline hover:no-underline"
                  >
                    {listing.channel_title ?? listing.channel_username}
                  </a>
                ) : (
                  listing.channel_title ?? '—'
                )}
              </p>
            )}
            {listing.channel_followers != null && listing.channel_followers > 0 && (
              <p className="mt-1 text-sm text-muted-foreground">
                {formatFollowers(listing.channel_followers)} followers
              </p>
            )}
            {listing.categories && listing.categories.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1">
                {listing.categories.map((c) => (
                  <Badge key={c} variant="outline" className="font-normal">
                    {c}
                  </Badge>
                ))}
              </div>
            )}
            {listing.description && (
              <p className="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">{listing.description}</p>
            )}
          </CardHeader>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-lg">Pricing</CardTitle>
          </CardHeader>
          <CardContent>
            {priceRowsForListing.length > 0 ? (
              <div className="grid grid-cols-2 gap-2">
                {priceRowsForListing.map((row, i) => (
                  <div
                    key={i}
                    className="flex items-center justify-between rounded-lg border border-border bg-muted/30 px-4 py-3"
                  >
                    <span className="text-sm text-muted-foreground">{formatPriceKey(row.duration)}</span>
                    <span className="font-semibold tabular-nums text-primary">{formatPriceValue(row.price)}</span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">—</p>
            )}
          </CardContent>
        </Card>

        {listing.channel_id != null && (
          <Link
            href={`/profile/channels/${listing.channel_id}`}
            className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-border bg-card px-4 py-3 text-sm font-medium hover:bg-accent"
          >
            <BarChart3 size={18} />
            View channel stats
          </Link>
        )}

        {deals.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Active Deals</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {deals.map((d) => (
                  <Link
                    key={d.id}
                    href={`/deals/${d.id}`}
                    className="flex cursor-pointer items-center justify-between rounded-md bg-muted/50 p-3 hover:bg-muted"
                  >
                    <div>
                      <p className="text-sm font-medium">Deal #{d.id}</p>
                      <p className="text-xs text-muted-foreground">
                        {formatPriceKey(String(d.duration))} – {formatPriceValue(d.price)}
                      </p>
                    </div>
                    <Badge>{getDealStatusLabel(d.status)}</Badge>
                  </Link>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {isOwner && (
          <div className="flex flex-col gap-2">
            {listing.status === 'active' ? (
              <Button
                variant="outline"
                className="w-full"
                disabled={actionLoading}
                onClick={async () => {
                  setActionLoading(true);
                  const res = await api<Listing>(`/api/v1/market/listings/${id}`, {
                    method: 'PATCH',
                    body: JSON.stringify({ status: 'inactive' }),
                  });
                  setActionLoading(false);
                  if (res.ok && res.data) setListing(res.data);
                }}
              >
                <PowerOff size={18} className="mr-2" />
                Deactivate listing
              </Button>
            ) : (
              <Button
                variant="outline"
                className="w-full"
                disabled={actionLoading}
                onClick={async () => {
                  setActionLoading(true);
                  const res = await api<Listing>(`/api/v1/market/listings/${id}`, {
                    method: 'PATCH',
                    body: JSON.stringify({ status: 'active' }),
                  });
                  setActionLoading(false);
                  if (res.ok && res.data) setListing(res.data);
                }}
              >
                <Power size={18} className="mr-2" />
                Activate listing
              </Button>
            )}
            <Button
              variant="outline"
              className="w-full text-destructive hover:bg-destructive/10 hover:text-destructive"
              disabled={actionLoading}
              onClick={async () => {
                if (!confirm('Delete this listing? This cannot be undone.')) return;
                setActionLoading(true);
                const res = await api<{ status: string }>(`/api/v1/market/listings/${id}`, { method: 'DELETE' });
                setActionLoading(false);
                if (res.ok) router.push('/my-listings');
                else alert(res.error_code || 'Failed to delete');
              }}
            >
              <Trash2 size={18} className="mr-2" />
              Delete listing
            </Button>
          </div>
        )}

        {listing.status === 'active' && !isOwner && (
          <Button className="w-full" size="lg" onClick={handleOpenCreateDeal}>
            <MessageCircle size={18} className="mr-2" />
            {listing.type === 'lessor' ? 'Apply to This Channel' : 'Contact Advertiser'}
          </Button>
        )}
      </div>

      {showCreateDealModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={() => !createDealSubmitting && setShowCreateDealModal(false)}
          role="dialog"
          aria-modal="true"
        >
          <div
            className="w-full max-w-md rounded-xl border border-border bg-card p-4 shadow-lg"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold">Create deal</h3>
              <button
                type="button"
                onClick={() => !createDealSubmitting && setShowCreateDealModal(false)}
                className="rounded p-1 text-muted-foreground hover:bg-accent"
                aria-label="Close"
              >
                <X size={20} />
              </button>
            </div>
            <div className="mt-4 space-y-3">
              {listing?.type === 'lessee' && (
                <div>
                  <Label className="text-sm text-muted-foreground">Your channel (where the ad will be posted)</Label>
                  <select
                    value={createDealChannelId ?? ''}
                    onChange={(e) => setCreateDealChannelId(e.target.value ? Number(e.target.value) : null)}
                    className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                    required
                  >
                    <option value="">Select a channel</option>
                    {myChannels.map((ch) => (
                      <option key={ch.id} value={ch.id}>
                        {ch.title ?? ch.username ?? `Channel #${ch.id}`}
                      </option>
                    ))}
                  </select>
                  {myChannels.length === 0 && (
                    <p className="mt-1 text-xs text-muted-foreground">Add a channel in Profile to use it here.</p>
                  )}
                </div>
              )}
              {priceRowsForListing.length > 1 && (
                <div>
                  <Label className="text-sm text-muted-foreground">Price option</Label>
                  <select
                    value={createDealPriceIndex}
                    onChange={(e) => setCreateDealPriceIndex(Number(e.target.value))}
                    className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                  >
                    {priceRowsForListing.map((row, i) => (
                      <option key={i} value={i}>
                        {formatPriceEntry(row.duration, row.price)}
                      </option>
                    ))}
                  </select>
                </div>
              )}
              <div>
                <Label htmlFor="deal-message" className="text-sm text-muted-foreground">
                  Post text
                </Label>
                <textarea
                  id="deal-message"
                  value={dealMessage}
                  onChange={(e) => setDealMessage(e.target.value)}
                  placeholder="You can type here anything you want, either draft post text either some of your ad plans. This can be changed any time before approvement."
                  rows={4}
                  className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                />
              </div>
              <div className="min-w-0">
                <Label htmlFor="deal-posted-at" className="text-sm text-muted-foreground">
                  Date and time of posting
                </Label>
                <input
                  id="deal-posted-at"
                  type="datetime-local"
                  value={dealPostedAt}
                  onChange={(e) => setDealPostedAt(e.target.value)}
                  className="mt-1 w-full min-w-0 max-w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                />
              </div>
            </div>
            <div className="mt-4 flex gap-2">
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => setShowCreateDealModal(false)}
                disabled={createDealSubmitting}
              >
                Cancel
              </Button>
              <Button className="flex-1" onClick={handleCreateDeal} disabled={createDealSubmitting}>
                {createDealSubmitting ? 'Creating…' : 'Create deal'}
              </Button>
            </div>
          </div>
        </div>
      )}
      </div>
  ) );

  return (
    <>
      <div className={loading ? 'opacity-0' : 'opacity-100'}>{content}</div>
      <LoadingScreen show={loading} />
    </>
  );
}
