'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { api, auth, setAuthToken } from '@/lib/api';
import { LISTING_CATEGORIES } from '@/lib/constants';
import type { Channel, Listing } from '@/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

type ListingType = 'lessor' | 'lessee';
type ListingStatus = 'active' | 'inactive';

interface PriceRow {
  duration: string; // e.g. "24" -> "24hr"
  price: string;    // input string
}

export default function CreateListingPage() {
  const router = useRouter();
  const [type, setType] = useState<ListingType>('lessor');
  const [status, setStatus] = useState<ListingStatus>('inactive');
  const [channelId, setChannelId] = useState<string>('');
  const [prices, setPrices] = useState<PriceRow[]>([{ duration: '24', price: '' }]);
  const [categories, setCategories] = useState<string[]>([]);
  const [description, setDescription] = useState('');
  const [channels, setChannels] = useState<Channel[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
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
    if (!authed) return;
    api<Channel[]>('/api/v1/market/my-channels').then((res) => {
      if (res.ok && res.data) setChannels(res.data);
    });
  }, [authed]);

  const addPriceRow = () => {
    setPrices((prev) => [...prev, { duration: '24', price: '' }]);
  };

  const removePriceRow = (index: number) => {
    setPrices((prev) => prev.filter((_, i) => i !== index));
  };

  const toggleCategory = (cat: string) => {
    setCategories((prev) =>
      prev.includes(cat) ? prev.filter((c) => c !== cat) : [...prev, cat]
    );
  };

  const updatePriceRow = (index: number, field: 'duration' | 'price', value: string) => {
    setPrices((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], [field]: value };
      return next;
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    const authRes = await auth();
    if (!authRes.ok || !authRes.data) {
      setError('Open from Telegram to create a listing.');
      return;
    }
    setAuthToken(authRes.data);

    const pricePairs: [string, number][] = [];
    for (const row of prices) {
      const dur = row.duration.trim();
      const num = parseInt(row.price.trim(), 10);
      if (!dur || isNaN(num) || num < 0) continue;
      const durationStr = /^\d+$/.test(dur) ? `${dur}hr` : dur.endsWith('hr') ? dur : `${dur}hr`;
      pricePairs.push([durationStr, num]);
    }
    if (pricePairs.length === 0) {
      setError('Add at least one price (e.g. 24 hours, 100).');
      return;
    }

    setSubmitting(true);
    const body: {
      type: ListingType;
      status: ListingStatus;
      channel_id?: number;
      prices: [string, number][];
      categories?: string[];
      description?: string;
    } = {
      type,
      status,
      prices: pricePairs,
      categories: categories.length ? categories : undefined,
      description: description.trim() || undefined,
    };
    if (type === 'lessor') {
      const channelNum = channelId ? parseInt(channelId, 10) : null;
      if (channelNum && !isNaN(channelNum)) body.channel_id = channelNum;
    }

    const res = await api<Listing>('/api/v1/market/listings', {
      method: 'POST',
      body: JSON.stringify(body),
    });
    setSubmitting(false);
    if (res.ok && res.data) {
      router.push(`/listings/${res.data.id}`);
      return;
    }
    setError(res.error_code || 'Failed to create listing');
  };

  if (authed === false) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <p className="text-center text-muted-foreground">
          Open from Telegram to create a listing.
        </p>
        <Link href="/profile" className="mt-4 inline-block text-sm text-primary">
          ← Back to profile
        </Link>
      </div>
    );
  }

  return (
    <div className="min-h-screen pb-20">
      <div className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur">
        <div className="mx-auto max-w-2xl px-4 py-4">
          <div className="flex items-center gap-4">
            <Link
              href="/profile"
              className="inline-flex size-9 items-center justify-center rounded-md hover:bg-accent hover:text-accent-foreground"
            >
              <ArrowLeft size={20} />
            </Link>
            <h1 className="text-xl font-bold">Create listing</h1>
          </div>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="mx-auto max-w-2xl px-4 py-4">
        {error && (
          <p className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {error}
          </p>
        )}

        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-base">Type & status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label className="text-sm text-muted-foreground">Type</Label>
              <div className="mt-1 flex gap-2">
                <button
                  type="button"
                  onClick={() => setType('lessor')}
                  className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium ${
                    type === 'lessor'
                      ? 'border-primary bg-primary text-primary-foreground'
                      : 'border-border bg-background'
                  }`}
                >
                  Offering (lessor)
                </button>
                <button
                  type="button"
                  onClick={() => setType('lessee')}
                  className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium ${
                    type === 'lessee'
                      ? 'border-primary bg-primary text-primary-foreground'
                      : 'border-border bg-background'
                  }`}
                >
                  Seeking (lessee)
                </button>
              </div>
            </div>
            <div>
              <Label className="text-sm text-muted-foreground">Status</Label>
              <div className="mt-1 flex gap-2">
                <button
                  type="button"
                  onClick={() => setStatus('active')}
                  className={`flex-1 rounded-md border px-3 py-2 text-sm ${
                    status === 'active' ? 'border-primary bg-primary text-primary-foreground' : 'border-border'
                  }`}
                >
                  Active
                </button>
                <button
                  type="button"
                  onClick={() => setStatus('inactive')}
                  className={`flex-1 rounded-md border px-3 py-2 text-sm ${
                    status === 'inactive' ? 'border-primary bg-primary text-primary-foreground' : 'border-border'
                  }`}
                >
                  Inactive
                </button>
              </div>
            </div>
            {type === 'lessor' && channels.length > 0 && (
              <div>
                <Label htmlFor="channel" className="text-sm text-muted-foreground">
                  Channel
                </Label>
                <select
                  id="channel"
                  value={channelId}
                  onChange={(e) => setChannelId(e.target.value)}
                  className="mt-1 flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                >
                  <option value="">None</option>
                  {channels.map((c) => (
                    <option key={c.id} value={String(c.id)}>
                      {c.title} {c.username ? `@${c.username}` : ''}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-base">Categories</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="mb-2 text-sm text-muted-foreground">Select one or more categories for your listing.</p>
            <div className="flex flex-wrap gap-2">
              {LISTING_CATEGORIES.map((cat) => (
                <button
                  key={cat}
                  type="button"
                  onClick={() => toggleCategory(cat)}
                  className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                    categories.includes(cat)
                      ? 'border-primary bg-primary text-primary-foreground'
                      : 'border-border bg-background hover:bg-muted'
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-base">Description</CardTitle>
          </CardHeader>
          <CardContent>
            <textarea
              placeholder="Describe your listing (optional)"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              className="w-full resize-y rounded-md border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
          </CardContent>
        </Card>

        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-base">Pricing</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {prices.map((row, index) => (
              <div key={index} className="flex gap-2">
                <div className="flex-1">
                  <Input
                    type="text"
                    placeholder="hours"
                    value={row.duration}
                    onChange={(e) => updatePriceRow(index, 'duration', e.target.value)}
                  />
                </div>
                <div className="flex-1">
                  <Input
                    type="number"
                    min={0}
                    placeholder="TON"
                    value={row.price}
                    onChange={(e) => updatePriceRow(index, 'price', e.target.value)}
                  />
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  onClick={() => removePriceRow(index)}
                  disabled={prices.length <= 1}
                >
                  −
                </Button>
              </div>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={addPriceRow}>
              + Add price
            </Button>
          </CardContent>
        </Card>

        <Button type="submit" className="w-full" disabled={submitting}>
          {submitting ? 'Creating…' : 'Create listing'}
        </Button>
      </form>
    </div>
  );
}
