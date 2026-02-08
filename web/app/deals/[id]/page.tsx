'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { api, auth, setAuthToken } from '@/lib/api';
import { getTelegramUser } from '@/lib/initData';
import { formatPriceKey, formatPriceValue, parseListingPrices, formatPriceEntry } from '@/lib/formatPrice';
import type { Deal, DealChat, Listing } from '@/types';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

type Tab = 'details' | 'chat';

function getDealMessage(details: Record<string, unknown> | undefined): string {
  if (!details || typeof details.message !== 'string') return '';
  return details.message;
}

function getDealPostedAt(details: Record<string, unknown> | undefined): string {
  if (!details || typeof details.posted_at !== 'string') return '';
  return details.posted_at;
}

function formatPostedAt(iso: string): string {
  if (!iso) return '';
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
  } catch {
    return iso;
  }
}

function isoToDatetimeLocal(iso: string): string {
  if (!iso) return '';
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return '';
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  } catch {
    return '';
  }
}

export default function DealDetailPage() {
  const params = useParams();
  const id = Number(params?.id);
  const [deal, setDeal] = useState<Deal | null>(null);
  const [listing, setListing] = useState<Listing | null>(null);
  const [messages, setMessages] = useState<DealChat[]>([]);
  const [tab, setTab] = useState<Tab>('details');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [draftMessage, setDraftMessage] = useState('');
  const [draftPostedAt, setDraftPostedAt] = useState('');
  const [draftPriceIndex, setDraftPriceIndex] = useState(0);
  const [draftSaving, setDraftSaving] = useState(false);
  const [draftPostedAtError, setDraftPostedAtError] = useState<string | null>(null);
  const [signing, setSigning] = useState(false);
  const [currentUserId, setCurrentUserId] = useState<number | null>(null);

  // Initial fetch + polling for deal (status, escrow, etc.)
  useEffect(() => {
    if (!id || Number.isNaN(id)) {
      setError('Invalid deal ID');
      setLoading(false);
      return;
    }
    let isMounted = true;
    const fetchDeal = () => {
      api<Deal>(`/api/v1/market/deals/${id}`)
        .then((res) => {
          if (!isMounted) return;
          if (res.ok && res.data) {
            setDeal(res.data);
            setError(null);
            const d = res.data.details as Record<string, unknown>;
            setDraftMessage(getDealMessage(d));
            setDraftPostedAt(isoToDatetimeLocal(getDealPostedAt(d)));
          } else setError(res.error_code || 'Not found');
        })
        .catch(() => {
          if (isMounted) setError('Network error');
        })
        .finally(() => {
          if (isMounted) setLoading(false);
        });
    };
    fetchDeal();
    const interval = setInterval(fetchDeal, 3000);
    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [id]);

  useEffect(() => {
    if (!deal?.listing_id) return;
    api<Listing>(`/api/v1/market/listings/${deal.listing_id}`).then((res) => {
      if (res.ok && res.data) {
        setListing(res.data);
        const rows = parseListingPrices(res.data.prices);
        const idx = rows.findIndex(
          (r) => parseInt(r.duration, 10) === Number(deal.duration) && r.price === deal.price
        );
        if (idx >= 0) setDraftPriceIndex(idx);
      }
    });
  }, [deal?.listing_id, deal?.duration, deal?.price]);

  useEffect(() => {
    const user = getTelegramUser();
    if (user?.id != null) setCurrentUserId(user.id);
  }, []);

  // Chat messages: initial fetch when opening chat tab + polling every 3s while on chat tab
  useEffect(() => {
    if (tab !== 'chat' || !id || Number.isNaN(id)) return;
    const token = typeof window !== 'undefined' && localStorage.getItem('ads_mrkt_jwt');
    if (!token) return;
    let isMounted = true;
    const fetchMessages = () => {
      api<DealChat[]>(`/api/v1/market/deals/${id}/messages`)
        .then((res) => {
          if (isMounted && res.ok && res.data) setMessages(res.data);
        })
        .catch(() => {});
    };
    fetchMessages();
    const interval = setInterval(fetchMessages, 3000);
    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [id, tab]);

  const handleSendChatMessage = async () => {
    const authRes = await auth();
    if (!authRes.ok || !authRes.data) {
      alert('Open from Telegram to send messages.');
      return;
    }
    setAuthToken(authRes.data);
    const res = await api<DealChat>(`/api/v1/market/deals/${id}/send-chat-message`, {
      method: 'POST',
    });
    if (res.ok && res.data) {
      setMessages((prev) => [...prev, res.data!]);
    }
  };

  if (loading)
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  if (error || !deal)
    return (
      <div className="mx-auto max-w-3xl px-4 py-8">
        <p className="text-destructive">{error || 'Not found'}</p>
        <Link href="/" className="mt-2 inline-block text-sm text-muted-foreground hover:text-foreground">
          ← Back
        </Link>
      </div>
    );

  const isLessor = currentUserId != null && deal.lessor_id === currentUserId;
  const isLessee = currentUserId != null && deal.lessee_id === currentUserId;
  const lessorSigned = Boolean(deal.lessor_signature);
  const lesseeSigned = Boolean(deal.lessee_signature);
  const canSignAsLessor = deal.status === 'draft' && isLessor && !lessorSigned;
  const canSignAsLessee = deal.status === 'draft' && isLessee && !lesseeSigned;

  const handleSignDeal = async () => {
    const authRes = await auth();
    if (!authRes.ok || !authRes.data) {
      alert('Open from Telegram to sign.');
      return;
    }
    setAuthToken(authRes.data);
    setSigning(true);
    const res = await api<Deal>(`/api/v1/market/deals/${id}/sign`, { method: 'POST' });
    setSigning(false);
    if (res.ok && res.data) setDeal(res.data);
    else alert(res.error_code || 'Failed to sign');
  };

  return (
    <div className="min-h-screen pb-20">
      <div className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur">
        <div className="mx-auto max-w-3xl px-4 py-4">
          <Link href="/deals" className="text-sm text-muted-foreground hover:text-foreground">
            ← Deals
          </Link>
          <h1 className="mt-1 text-2xl font-bold">Deal #{deal.id}</h1>
        </div>
      </div>

      <div className="mx-auto max-w-3xl px-4 py-4">
        <Tabs value={tab} onValueChange={(v) => setTab(v as Tab)} className="w-full">
          <TabsList className="mb-4 grid w-full grid-cols-2">
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="chat">Chat history</TabsTrigger>
          </TabsList>

          <TabsContent value="details">
            <Card>
              <CardContent className="space-y-2 p-4">
                <p className="text-sm">
                  <strong>Status:</strong> {deal.status}
                </p>
                <div className="flex flex-wrap gap-3 text-sm">
                  <span className={lessorSigned ? 'text-muted-foreground' : ''}>
                    Lessor: {lessorSigned ? '✓ Signed' : 'Pending'}
                  </span>
                  <span className={lesseeSigned ? 'text-muted-foreground' : ''}>
                    Lessee: {lesseeSigned ? '✓ Signed' : 'Pending'}
                  </span>
                </div>
                {(canSignAsLessor || canSignAsLessee) && (
                  <Button
                    size="sm"
                    onClick={handleSignDeal}
                    disabled={signing}
                  >
                    {signing ? 'Signing…' : canSignAsLessor ? 'Sign as lessor' : 'Sign as lessee'}
                  </Button>
                )}
                {deal.status === 'waiting_escrow_deposit' && (
                  <div className="rounded-md border border-border bg-muted/30 p-3 text-sm">
                    {isLessor && (
                      <p className="text-muted-foreground">Waiting for lessee escrow deposit.</p>
                    )}
                    {isLessee && deal.escrow_address != null && (
                      <div>
                        <p className="font-medium mb-1">Deposit to escrow</p>
                        <p className="text-muted-foreground">Amount: {formatPriceValue(deal.price)}</p>
                        <p className="mt-1 break-all font-mono text-xs">{deal.escrow_address}</p>
                      </div>
                    )}
                  </div>
                )}
                <p className="text-sm">
                  <strong>Type:</strong> {deal.type}
                </p>
                <p className="text-sm">
                  <strong>Duration:</strong> {formatPriceKey(String(deal.duration))}
                </p>
                <p className="text-sm">
                  <strong>Price:</strong> {formatPriceValue(deal.price)}
                </p>
                <p className="text-sm">
                  <strong>Listing:</strong>{' '}
                  <Link
                    href={`/listings/${deal.listing_id}`}
                    className="text-primary underline hover:no-underline"
                  >
                    #{deal.listing_id}
                  </Link>
                </p>
                {deal.status !== 'waiting_escrow_deposit' && deal.escrow_address && (
                  <p className="text-sm">
                    <strong>Escrow:</strong> {deal.escrow_address}
                  </p>
                )}
                {deal.escrow_release_time && (
                  <p className="text-sm">
                    <strong>Release:</strong> {new Date(deal.escrow_release_time).toLocaleString()}
                  </p>
                )}
                {(getDealMessage(deal.details as Record<string, unknown>) || getDealPostedAt(deal.details as Record<string, unknown>)) && (
                  <div className="mt-2">
                    <strong className="text-sm">Post text</strong>
                    {getDealMessage(deal.details as Record<string, unknown>) && (
                      <p className="mt-1 whitespace-pre-wrap rounded-md border border-border bg-muted/50 p-2 text-sm">
                        {getDealMessage(deal.details as Record<string, unknown>)}
                      </p>
                    )}
                    {getDealPostedAt(deal.details as Record<string, unknown>) && (
                      <p className="mt-1 text-xs text-muted-foreground">
                        Posted: {formatPostedAt(getDealPostedAt(deal.details as Record<string, unknown>))}
                      </p>
                    )}
                  </div>
                )}
                {deal.status === 'draft' && listing && (() => {
                  const priceRows = parseListingPrices(listing.prices);
                  if (priceRows.length === 0) return null;
                  const row = priceRows[draftPriceIndex] ?? priceRows[0];
                  return (
                    <div className="mt-4 space-y-3 border-t border-border pt-4">
                      <p className="text-sm font-medium">Edit draft</p>
                      {priceRows.length > 1 && (
                        <div>
                          <Label className="text-xs text-muted-foreground">Price option</Label>
                          <select
                            value={draftPriceIndex}
                            onChange={(e) => setDraftPriceIndex(Number(e.target.value))}
                            className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                          >
                            {priceRows.map((r, i) => (
                              <option key={i} value={i}>{formatPriceEntry(r.duration, r.price)}</option>
                            ))}
                          </select>
                        </div>
                      )}
                      <div>
                        <Label className="text-xs text-muted-foreground">Post text</Label>
                        <textarea
                          value={draftMessage}
                          onChange={(e) => setDraftMessage(e.target.value)}
                          placeholder="Text of the post..."
                          rows={3}
                          className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                        />
                      </div>
                      <div>
                        <Label className="text-xs text-muted-foreground">Date and time of posting</Label>
                        <input
                          type="datetime-local"
                          value={draftPostedAt}
                          onChange={(e) => {
                            setDraftPostedAt(e.target.value);
                            setDraftPostedAtError(null);
                          }}
                          className="mt-1 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                        />
                        {draftPostedAtError && (
                          <p className="mt-1 text-xs text-destructive">{draftPostedAtError}</p>
                        )}
                      </div>
                      <Button
                        size="sm"
                        disabled={draftSaving}
                        onClick={async () => {
                          let postedAtVal: string | undefined;
                          if (draftPostedAt.trim()) {
                            try {
                              const d = new Date(draftPostedAt.trim());
                              if (Number.isNaN(d.getTime())) {
                                setDraftPostedAtError('Invalid date and time');
                                return;
                              }
                              postedAtVal = d.toISOString();
                            } catch {
                              setDraftPostedAtError('Invalid date and time');
                              return;
                            }
                          }
                          setDraftPostedAtError(null);
                          const authRes = await auth();
                          if (!authRes.ok || !authRes.data) return;
                          setAuthToken(authRes.data);
                          const r = priceRows[draftPriceIndex] ?? priceRows[0];
                          const type = r.duration + 'hr';
                          const duration = parseInt(r.duration, 10) || 24;
                          setDraftSaving(true);
                          const res = await api<Deal>(`/api/v1/market/deals/${id}`, {
                            method: 'PATCH',
                            body: JSON.stringify({
                              type,
                              duration,
                              price: r.price,
                              details: {
                                message: draftMessage.trim() || undefined,
                                posted_at: postedAtVal,
                              },
                            }),
                          });
                          setDraftSaving(false);
                          if (res.ok && res.data) setDeal(res.data);
                          else alert(res.error_code || 'Failed to update');
                        }}
                      >
                        {draftSaving ? 'Saving…' : 'Save draft'}
                      </Button>
                    </div>
                  );
                })()}
                <p className="pt-2 text-xs text-muted-foreground">
                  Updated {new Date(deal.updated_at).toLocaleString()}
                </p>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="chat">
            <div className="space-y-2">
              {messages.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">No messages yet.</p>
              ) : (
                messages.map((m, i) => {
                  const isFromMe = currentUserId != null && m.reply_to_chat_id !== currentUserId;
                  const otherLabel = isLessor ? 'Lessee' : isLessee ? 'Lessor' : 'Other';
                  const authorLabel = isFromMe ? 'Me' : otherLabel;
                  return (
                    <Card key={i}>
                      <CardContent className="p-3">
                        <p className="text-xs font-medium text-muted-foreground">{authorLabel}</p>
                        {m.replied_message != null && (
                          <p className="mt-1 text-sm whitespace-pre-wrap">{m.replied_message}</p>
                        )}
                        <p className="mt-1 text-xs text-muted-foreground">
                          {new Date(m.created_at).toLocaleString()}
                        </p>
                      </CardContent>
                    </Card>
                  );
                })
              )}
            </div>
            <Button className="mt-4 w-full" onClick={handleSendChatMessage}>
              Send chat invite
            </Button>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
