'use client';

import { useEffect, useState, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useTonAddress, useTonWallet, useTonConnectUI } from '@tonconnect/ui-react';
import { api, auth, setAuthToken } from '@/lib/api';
import { useTelegramBackButton } from '@/lib/telegram';
import { getTelegramUser } from '@/lib/initData';
import { formatPriceKey, formatPriceValue, parseListingPrices, formatPriceEntry } from '@/lib/formatPrice';
import { toFriendlyAddress, formatAddressForDisplay, truncateAddressDisplay, addressesEqual } from '@/lib/tonAddress';
import type { Deal, DealChat, Listing } from '@/types';
import { getDealStatusLabel } from '@/types';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { LoadingScreen } from '@/components/LoadingScreen';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { BarChart3 } from 'lucide-react';

/** TON logo (official style). White in SVG; use currentColor so it matches text (black in light theme, white in dark). */
function TonLogoIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 237 237"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-hidden
    >
      <path d="M118.204 0.000292436C183.486 0.000292436 236.408 52.9224 236.408 118.205C236.408 183.487 183.486 236.408 118.204 236.408C52.9216 236.408 0.000184007 183.487 0 118.205C0 52.9225 52.9215 0.000452012 118.204 0.000292436ZM74.1011 62.1965C57.6799 62.1965 47.268 79.912 55.5308 94.2347L109.964 188.582C113.619 194.922 122.781 194.922 126.436 188.582L180.88 94.2347C189.132 79.9343 178.72 62.1966 162.31 62.1965H74.1011ZM162.288 78.8412C166.031 78.8412 168.234 82.8121 166.45 85.9075L137.856 137.091L137.851 137.099L126.506 159.046V78.8412H162.288ZM109.872 78.8517V159.024L98.5376 137.088L98.5334 137.08L69.9294 85.9215L69.8468 85.7725C68.2134 82.6997 70.405 78.8517 74.0899 78.8517H109.872Z" />
    </svg>
  );
}

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
  const router = useRouter();
  const id = Number(params?.id);
  useTelegramBackButton(() => router.back());
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
  const [rejecting, setRejecting] = useState(false);
  const [depositing, setDepositing] = useState(false);
  const [depositError, setDepositError] = useState<string | null>(null);
  const [currentUserId, setCurrentUserId] = useState<number | null>(null);
  const walletSyncedRef = useRef(false);
  const dealPayoutSyncedRef = useRef(false);

  const wallet = useTonWallet();
  const rawAddress = useTonAddress(false);
  const [tonConnectUI] = useTonConnectUI();

  // Sync connected wallet to backend (raw format) so user can sign deals.
  useEffect(() => {
    if (!rawAddress || walletSyncedRef.current) return;
    (async () => {
      const authRes = await auth();
      if (!authRes.ok || !authRes.data) return;
      setAuthToken(authRes.data);
      const res = await api<{ status: string }>('/api/v1/market/me/wallet', {
        method: 'PUT',
        body: JSON.stringify({ wallet_address: rawAddress }),
      });
      if (res.ok) walletSyncedRef.current = true;
    })();
  }, [rawAddress]);

  // Set this deal's payout address when wallet connected and user is lessor or lessee.
  useEffect(() => {
    if (!rawAddress || !deal || dealPayoutSyncedRef.current || !id) return;
    const isSide = (currentUserId === deal.lessor_id || currentUserId === deal.lessee_id) && deal.status === 'draft';
    if (!isSide) return;
    (async () => {
      const authRes = await auth();
      if (!authRes.ok || !authRes.data) return;
      setAuthToken(authRes.data);
      const res = await api<Deal>(`/api/v1/market/deals/${id}/payout-address`, {
        method: 'PUT',
        body: JSON.stringify({ wallet_address: rawAddress }),
      });
      if (res.ok && res.data) {
        dealPayoutSyncedRef.current = true;
        setDeal(res.data);
      }
    })();
  }, [rawAddress, deal, id, currentUserId]);

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

  const isLessor = deal != null && currentUserId != null && deal.lessor_id === currentUserId;
  const isLessee = deal != null && currentUserId != null && deal.lessee_id === currentUserId;
  const lessorSigned = Boolean(deal?.lessor_signature);
  const lesseeSigned = Boolean(deal?.lessee_signature);
  const canSignAsLessor = (deal?.status === 'draft') && isLessor && !lessorSigned;
  const canSignAsLessee = (deal?.status === 'draft') && isLessee && !lesseeSigned;
  const bothPayoutsSet = Boolean(deal?.lessor_payout_address && deal?.lessee_payout_address);
  const needsWalletToSign = (canSignAsLessor || canSignAsLessee) && !wallet;
  const canSignNow = (canSignAsLessor || canSignAsLessee) && wallet && bothPayoutsSet;

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

  const handleRejectDeal = async () => {
    const authRes = await auth();
    if (!authRes.ok || !authRes.data) {
      alert('Open from Telegram to reject.');
      return;
    }
    setAuthToken(authRes.data);
    if (!confirm('Reject this deal? This cannot be undone.')) return;
    setRejecting(true);
    const res = await api<Deal>(`/api/v1/market/deals/${id}/reject`, { method: 'POST' });
    setRejecting(false);
    if (res.ok && res.data) setDeal(res.data);
    else alert(res.error_code || 'Failed to reject');
  };

  const handleDepositEscrow = async () => {
    if (!deal?.escrow_address || deal.escrow_amount == null || deal.escrow_amount <= 0) {
      setDepositError('Escrow details not ready');
      return;
    }
    if (!wallet) {
      setDepositError('Connect your wallet first');
      return;
    }
    setDepositError(null);
    setDepositing(true);
    try {
      const escrowFriendly = toFriendlyAddress(deal!.escrow_address, false);
      await tonConnectUI.sendTransaction({
        validUntil: Math.floor(Date.now() / 1000) + 300,
        messages: [
          {
            address: escrowFriendly,
            amount: String(deal!.escrow_amount),
          },
        ],
      });
      setDeal((prev) => prev ? { ...prev } : null);
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Transaction failed';
      setDepositError(msg);
    } finally {
      setDepositing(false);
    }
  };

  return (
    <>
      <div className={loading ? 'opacity-0' : 'opacity-100'}>
        {loading ? (
          <div className="min-h-screen" aria-hidden />
        ) : (error || !deal) ? (
          <div className="page-with-nav">
            <PageTopSpacer />
            <div className="mx-auto max-w-3xl px-4 py-8">
              <p className="text-destructive">{error || 'Not found'}</p>
            </div>
          </div>
        ) : deal ? (
          <div className="page-with-nav">
      <PageTopSpacer />
      <div className="mx-auto max-w-3xl px-4 py-4">
        <Tabs value={tab} onValueChange={(v) => setTab(v as Tab)} className="w-full">
          <TabsList className="mb-4 grid w-full grid-cols-2">
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="chat">Chat history</TabsTrigger>
          </TabsList>

          <TabsContent value="details">
            {/* Wallet: above details card, minimal UI */}
            {(isLessor || isLessee) && (
              <div className="mb-4 flex w-full max-w-md flex-col gap-1">
                <div className="flex w-full items-center justify-between gap-2">
                  {!wallet ? (
                    <>
                      <span className="text-sm text-muted-foreground">Wallet</span>
                      <button
                        type="button"
                        onClick={() => tonConnectUI.openModal()}
                        className="shrink-0 inline-flex items-center gap-1.5 text-sm text-primary hover:underline"
                      >
                        <span>Connect</span>
                        <TonLogoIcon className="h-[1em] w-auto shrink-0" />
                      </button>
                    </>
                  ) : (
                    <>
                      <span className="font-mono text-sm tabular-nums">{truncateAddressDisplay(rawAddress || '')}</span>
                      <button
                        type="button"
                        onClick={async () => {
                          const authRes = await auth();
                          if (authRes.ok && authRes.data) {
                            setAuthToken(authRes.data);
                            await api<{ status: string }>('/api/v1/market/me/wallet', { method: 'DELETE' });
                          }
                          walletSyncedRef.current = false;
                          dealPayoutSyncedRef.current = false;
                          tonConnectUI.disconnect();
                        }}
                        className="shrink-0 text-sm text-red-600 hover:text-red-700 hover:underline"
                      >
                        Disconnect
                      </button>
                    </>
                  )}
                </div>
                {deal.status === 'draft' && !wallet && (
                  <p className="text-center text-xs text-muted-foreground">You need to connect wallet to make a deal.</p>
                )}
              </div>
            )}

            <Card>
              <CardContent className="space-y-2 p-4">
                <p className="text-sm">
                  <strong>Status:</strong> {getDealStatusLabel(deal.status)}
                </p>
                <div className="flex flex-wrap gap-3 text-sm">
                  <span className={lessorSigned ? 'text-muted-foreground' : ''}>
                    Lessor: {lessorSigned ? '✓ Signed' : 'Pending'}
                  </span>
                  <span className={lesseeSigned ? 'text-muted-foreground' : ''}>
                    Lessee: {lesseeSigned ? '✓ Signed' : 'Pending'}
                  </span>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {(canSignAsLessor || canSignAsLessee) && (
                    <>
                      {!bothPayoutsSet && (
                        <span className="text-sm text-muted-foreground">Waiting for both parties to set payout address.</span>
                      )}
                      {canSignNow && (
                        <Button
                          size="sm"
                          onClick={handleSignDeal}
                          disabled={signing || rejecting}
                        >
                          {signing ? 'Signing…' : canSignAsLessor ? 'Sign as lessor' : 'Sign as lessee'}
                        </Button>
                      )}
                    </>
                  )}
                </div>
                {deal.status === 'waiting_escrow_deposit' && (
                  <div className="rounded-md border border-border bg-muted/30 p-3 text-sm">
                    {isLessor && (
                      <p className="text-muted-foreground">Waiting for lessee escrow deposit.</p>
                    )}
                    {isLessee && deal.escrow_address != null && (
                      <div className="space-y-2">
                        <p className="font-medium">Deposit to escrow</p>
                        <p className="text-muted-foreground">
                          Amount: {deal.escrow_amount != null
                            ? `${(deal.escrow_amount / 1e9).toFixed(4)} TON`
                            : formatPriceValue(deal.price)}
                        </p>
                        <p className="break-all font-mono text-xs">{formatAddressForDisplay(deal.escrow_address)}</p>
                        {!wallet ? (
                          <p className="py-3 text-center text-sm text-muted-foreground">Connect wallet to proceed.</p>
                        ) : !deal.lessee_payout_address || !addressesEqual(rawAddress ?? '', deal.lessee_payout_address) ? (
                          <p className="py-3 text-center text-sm text-muted-foreground">Connect original wallet to proceed.</p>
                        ) : (
                          <Button
                            size="sm"
                            onClick={handleDepositEscrow}
                            disabled={depositing || !deal.escrow_amount || deal.escrow_amount <= 0}
                          >
                            {depositing ? 'Opening wallet…' : 'Deposit via wallet'}
                          </Button>
                        )}
                        {depositError && (
                          <p className="text-xs text-destructive">{depositError}</p>
                        )}
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
                {deal.status === 'draft' && isLessee && (deal.channel_id ?? listing?.channel_id) != null && (
                  <Link
                    href={`/profile/channels/${deal.channel_id ?? listing?.channel_id}`}
                    className="inline-flex items-center justify-center gap-2 rounded-lg border border-border bg-card px-4 py-2 text-sm font-medium hover:bg-accent"
                  >
                    <BarChart3 size={18} />
                    View channel stats
                  </Link>
                )}
                {deal.status !== 'waiting_escrow_deposit' && deal.escrow_address && (
                  <p className="text-sm">
                    <strong>Escrow:</strong> {formatAddressForDisplay(deal.escrow_address)}
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
                      <div className="flex flex-wrap items-center gap-2">
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
                        <Button
                          size="sm"
                          variant="outline"
                          className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                          onClick={handleRejectDeal}
                          disabled={signing || rejecting || draftSaving}
                        >
                          {rejecting ? 'Rejecting…' : 'Reject deal'}
                        </Button>
                      </div>
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
                    <Card key={i} className="py-0">
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
        ) : null}
      </div>
      <LoadingScreen show={loading} />
    </>
  );
}
