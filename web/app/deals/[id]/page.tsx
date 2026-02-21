'use client';

import React, { useEffect, useState, useRef } from 'react';
import { createPortal } from 'react-dom';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useTonAddress, useTonWallet, useTonConnectUI } from '@tonconnect/ui-react';
import { api, auth, setAuthToken } from '@/lib/api';
import { useTelegramBackButton, openTelegramLink } from '@/lib/telegram';
import { getTelegramUser } from '@/lib/initData';
import { formatPriceKey, formatPriceValue, parseListingPrices, formatPriceEntry } from '@/lib/formatPrice';
import { toFriendlyAddress, formatAddressForDisplay, truncateAddressDisplay, addressesEqual } from '@/lib/tonAddress';
import type { Deal, Listing } from '@/types';
import { getDealStatusLabel } from '@/types';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { LoadingScreen } from '@/components/LoadingScreen';
import { BarChart3, MessageCircle, FileEdit, FileCheck, Wallet, CircleCheck, Play, Send, CheckCircle2, Clock, XCircle } from 'lucide-react';
import type { DealStatus } from '@/types';
import { HandshakeDealSign } from '@/components/HandshakeDealSign';

export type RoadmapAlignment = 'left' | 'middle' | 'right';

/** Per-current-status segment and alignment. alignment = where current status sits in the bar. */
function getRoadmapSegment(current: DealStatus): { segment: DealStatus[]; alignment: RoadmapAlignment } {
  const segment: DealStatus[] = (() => {
    switch (current) {
      case 'draft':
        return ['draft', 'approved', 'waiting_escrow_deposit'];
      case 'approved':
        return ['draft', 'approved', 'waiting_escrow_deposit'];
      case 'waiting_escrow_deposit':
        return ['approved', 'waiting_escrow_deposit', 'escrow_deposit_confirmed'];
      case 'escrow_deposit_confirmed':
        return ['waiting_escrow_deposit', 'escrow_deposit_confirmed', 'in_progress'];
      case 'in_progress':
        return ['escrow_deposit_confirmed', 'in_progress', 'waiting_escrow_release'];
      case 'waiting_escrow_release':
        return ['in_progress', 'waiting_escrow_release', 'escrow_release_confirmed'];
      case 'escrow_release_confirmed':
        return ['waiting_escrow_release', 'escrow_release_confirmed', 'completed'];
      case 'completed':
        return ['waiting_escrow_release', 'escrow_release_confirmed', 'completed'];
      case 'waiting_escrow_refund':
        return ['in_progress', 'waiting_escrow_refund', 'escrow_refund_confirmed'];
      case 'escrow_refund_confirmed':
        return ['waiting_escrow_refund', 'escrow_refund_confirmed', 'completed'];
      case 'expired':
        return ['approved', 'waiting_escrow_deposit', 'expired'];
      case 'rejected':
        return ['draft', 'rejected'];
      default:
        return ['draft', 'approved', 'waiting_escrow_deposit'];
    }
  })();
  const idx = segment.indexOf(current);
  const alignment: RoadmapAlignment =
    idx <= 0 ? 'left' : idx >= segment.length - 1 ? 'right' : 'middle';
  return { segment, alignment };
}

const DEAL_STATUS_ICON: Record<DealStatus, typeof FileEdit> = {
  draft: FileEdit,
  approved: FileCheck,
  waiting_escrow_deposit: Wallet,
  escrow_deposit_confirmed: CircleCheck,
  in_progress: Play,
  waiting_escrow_release: Send,
  escrow_release_confirmed: CircleCheck,
  completed: CheckCircle2,
  waiting_escrow_refund: Wallet,
  escrow_refund_confirmed: CircleCheck,
  expired: Clock,
  rejected: XCircle,
};

function DealStatusRoadmap({
  currentStatus,
}: {
  currentStatus: DealStatus | string;
}) {
  const [tooltip, setTooltip] = useState<{ label: string; x: number; y: number } | null>(null);
  const current = currentStatus as DealStatus;
  const { segment, alignment } = getRoadmapSegment(current);
  const segmentCurrentIndex = segment.indexOf(current);
  const effectiveIndex = segmentCurrentIndex >= 0 ? segmentCurrentIndex : 0;

  const justify =
    alignment === 'left' ? 'justify-start' : alignment === 'right' ? 'justify-end' : 'justify-center';

  const handleTap = (status: DealStatus, event: React.MouseEvent<HTMLButtonElement>) => {
    if (status === current) return;
    const label = getDealStatusLabel(status);
    const rect = (event.target as HTMLElement).closest('button')?.getBoundingClientRect();
    if (!rect) return;
    setTooltip({
      label,
      x: rect.left + rect.width / 2,
      y: rect.top,
    });
    setTimeout(() => setTooltip(null), 2500);
  };

  const showLeftLine = effectiveIndex > 0;
  // Always show right line + three dots (when 2 statuses, the "third" is the straight line then dots)
  const showRightLine = segment.length > 0;

  // viewBox 0 0 100 24: line thickness = dot (4 units); inflow = bigger bulge at circle (12 units), taper over 10 units
  const LINE_TOP = 10;
  const LINE_BOTTOM = 14;
  const INFLOW_TOP = 6;
  const INFLOW_BOTTOM = 18;
  const INFLOW_UNITS = 10;
  const CAP_R = 2; // round cap at line end (before dots), radius 2

  /** Line as thick as dots, thickening at both ends (water-drop inflow). Optional gradient: current → future (muted). Future-only segments use muted fill. */
  const Connector = ({
    colorClass,
    className = '',
    useGradient = false,
    gradientId = 'connector-grad',
    useMutedFill = false,
  }: {
    colorClass: string;
    className?: string;
    useGradient?: boolean;
    gradientId?: string;
    useMutedFill?: boolean;
  }) => {
    const d = [
      `M 0 ${INFLOW_TOP} L ${INFLOW_UNITS} ${LINE_TOP} L ${100 - INFLOW_UNITS} ${LINE_TOP} L 100 ${INFLOW_TOP}`,
      `L 100 ${INFLOW_BOTTOM} L ${100 - INFLOW_UNITS} ${LINE_BOTTOM} L ${INFLOW_UNITS} ${LINE_BOTTOM} L 0 ${INFLOW_BOTTOM} Z`,
    ].join(' ');
    const pathFill =
      useGradient ? `url(#${gradientId})` : useMutedFill ? 'var(--muted)' : undefined;
    return (
      <div
        className={`flex min-w-0 flex-1 items-center ${className}`}
        style={{ marginLeft: '-4px', marginRight: '-4px', minWidth: 20 }}
        aria-hidden
      >
        <svg
          viewBox="0 0 100 24"
          preserveAspectRatio="none"
          className="h-6 w-full"
          style={{ overflow: 'visible' }}
        >
          {useGradient && (
            <defs>
              <linearGradient id={gradientId} x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stopColor="var(--primary)" />
                <stop offset="35%" stopColor="var(--primary)" />
                <stop offset="65%" stopColor="var(--muted)" />
                <stop offset="100%" stopColor="var(--muted)" />
              </linearGradient>
            </defs>
          )}
          <path
            d={d}
            fill={pathFill ?? 'currentColor'}
            className={pathFill ? undefined : colorClass}
          />
        </svg>
      </div>
    );
  };

  /** Line with inflow at right (meets first circle); rounded cap at left (before dots). */
  const LeftLineWithFlare = () => {
    const d = [
      `M ${CAP_R} ${LINE_TOP} L ${100 - INFLOW_UNITS} ${LINE_TOP} L 100 ${INFLOW_TOP} L 100 ${INFLOW_BOTTOM} L ${100 - INFLOW_UNITS} ${LINE_BOTTOM} L ${CAP_R} ${LINE_BOTTOM}`,
      `A ${CAP_R} ${CAP_R} 0 0 1 ${CAP_R} ${LINE_TOP} Z`,
    ].join(' ');
    return (
      <div className="flex min-w-0 flex-1 items-center justify-end" style={{ marginRight: '-4px' }} aria-hidden>
        <svg
          viewBox="0 0 100 24"
          preserveAspectRatio="none"
          className="h-6 min-w-[12px] flex-1"
          style={{ overflow: 'visible' }}
        >
          <path d={d} fill="currentColor" className="text-foreground dark:text-white" />
        </svg>
      </div>
    );
  };

  /** Line with inflow at left (meets last circle); rounded cap at right (before dots). */
  const RightLineWithFlare = () => {
    const d = [
      `M 0 ${INFLOW_TOP} L 0 ${INFLOW_BOTTOM} L ${INFLOW_UNITS} ${LINE_BOTTOM} L ${100 - CAP_R} ${LINE_BOTTOM}`,
      `A ${CAP_R} ${CAP_R} 0 0 0 ${100 - CAP_R} ${LINE_TOP} L ${INFLOW_UNITS} ${LINE_TOP} Z`,
    ].join(' ');
    return (
      <div className="flex min-w-0 flex-1 items-center justify-start" style={{ marginLeft: '-4px' }} aria-hidden>
        <svg
          viewBox="0 0 100 24"
          preserveAspectRatio="none"
          className="h-6 min-w-[12px] flex-1"
          style={{ overflow: 'visible' }}
        >
          <path d={d} fill="var(--muted)" />
        </svg>
      </div>
    );
  };

  return (
    <div className="relative w-full py-4">
      <div className={`relative z-10 flex items-center ${justify} gap-0`}>
        {/* Left: three dots + line (always when there is a previous step) */}
        {showLeftLine && (
          <div className="flex min-w-0 flex-1 items-center justify-end gap-0">
            <span
              className="flex items-end gap-0.5 pr-1 text-[8px] leading-none text-foreground dark:text-white"
              aria-hidden
            >
              <span className="inline-block h-1 w-1 rounded-full bg-current opacity-80" />
              <span className="inline-block h-1 w-1.5 rounded-full bg-current opacity-80" />
              <span className="inline-block h-1 w-2 rounded-full bg-current opacity-80" />
            </span>
            <LeftLineWithFlare />
          </div>
        )}

        {/* Circles and connectors from segment */}
        {segment.map((status, idx) => {
          const Icon = DEAL_STATUS_ICON[status];
          const label = getDealStatusLabel(status);
          const isCurrent = status === current;
          const isPast = effectiveIndex >= 0 && idx < effectiveIndex;
          const isPastOrCurrent = isCurrent || isPast;
          const nextIdx = idx + 1;
          const nextIsPastOrCurrent = nextIdx <= effectiveIndex;
          const isFutureSegment = !nextIsPastOrCurrent;
          const connectorColor =
            nextIsPastOrCurrent
              ? 'text-foreground dark:text-white'
              : 'text-muted-foreground/70';
          const isCurrentToNext = isCurrent && nextIdx > effectiveIndex;
          return (
            <React.Fragment key={status}>
              <div className="relative z-10 flex flex-shrink-0 flex-col items-center">
                {isCurrent && (
                  <span
                    className="absolute bottom-full left-1/2 z-20 mb-1 -translate-x-1/2 whitespace-nowrap text-sm font-semibold text-primary dark:text-white"
                    style={{ pointerEvents: 'none' }}
                  >
                    {label}
                  </span>
                )}
                <button
                  type="button"
                  onClick={(e) => handleTap(status, e)}
                  title={isCurrent ? undefined : label}
                  className={`relative flex flex-shrink-0 items-center justify-center rounded-full bg-transparent transition-colors hover:opacity-90 ${isCurrent ? 'h-11 w-11' : 'h-10 w-10'}`}
                >
                  <span
                    className={
                      'flex items-center justify-center rounded-full border-0 transition-colors ' +
                      (isCurrent ? 'h-10 w-10' : 'h-9 w-9') +
                      ' ' +
                      (isPastOrCurrent
                        ? 'bg-primary text-primary-foreground dark:bg-white dark:text-black'
                        : 'bg-muted text-muted-foreground dark:bg-muted dark:text-muted-foreground')
                    }
                  >
                    <Icon size={isCurrent ? 20 : 18} />
                  </span>
                </button>
              </div>
              {idx < segment.length - 1 ? (
                <Connector
                  colorClass={connectorColor}
                  className="flex-1 min-w-[24px] max-w-[80px]"
                  useGradient={isCurrentToNext}
                  gradientId={`connector-grad-${status}-${idx}`}
                  useMutedFill={isFutureSegment && !isCurrentToNext}
                />
              ) : showRightLine ? (
                <Connector
                  colorClass="text-muted-foreground/70"
                  className="flex-1 min-w-[24px] max-w-[80px]"
                  useMutedFill
                />
              ) : null}
            </React.Fragment>
          );
        })}

        {/* Right: line with rounded end + three dots (same color as line: muted) */}
        {showRightLine && (
          <div className="flex min-w-0 flex-1 items-center justify-start gap-0">
            <RightLineWithFlare />
            <span
              className="flex items-end gap-0.5 pl-1 text-[8px] leading-none"
              aria-hidden
            >
              <span className="inline-block h-1 w-2 rounded-full bg-[var(--muted)] opacity-80" />
              <span className="inline-block h-1 w-1.5 rounded-full bg-[var(--muted)] opacity-80" />
              <span className="inline-block h-1 w-1 rounded-full bg-[var(--muted)] opacity-80" />
            </span>
          </div>
        )}
      </div>

      {/* Tooltip: one shape (rounded rect + small arrow), centered on tap so arrow points at status; text single line */}
      {tooltip &&
        typeof document !== 'undefined' &&
        document.body &&
        createPortal(
          <div
            className="fixed z-[100] flex flex-col items-center"
            style={{
              left: tooltip.x,
              top: tooltip.y,
              transform: 'translate(-50%, calc(-100% - 10px))',
            }}
          >
            <div className="relative flex px-3 py-2">
              <svg
                className="absolute inset-0 h-full w-full"
                viewBox="0 0 120 40"
                preserveAspectRatio="none"
                style={{ filter: 'drop-shadow(0 1px 3px rgba(0,0,0,0.12))' }}
              >
                {/* Rounded rect + small downward arrow (~3x smaller); border continues around arrow */}
                <path
                  d="M 12 0 L 108 0 Q 120 0 120 12 L 120 30 Q 120 36 114 36 L 78 36 L 60 40 L 42 36 L 6 36 Q 0 36 0 30 L 0 12 Q 0 0 12 0 Z"
                  fill="var(--popover)"
                  stroke="var(--border)"
                  strokeWidth="1"
                  strokeLinejoin="round"
                />
              </svg>
              <span className="relative z-10 whitespace-nowrap text-sm font-medium text-popover-foreground">
                {tooltip.label}
              </span>
            </div>
          </div>,
          document.body
        )}
    </div>
  );
}

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

function formatTimeLeftMs(ms: number): string {
  if (ms <= 0) return '0:00';
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${String(s).padStart(2, '0')}`;
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
  const [loading, setLoading] = useState(true);
  const [chatLinkLoading, setChatLinkLoading] = useState(false);
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
  const draftEditorsSyncedRef = useRef(false);

  const wallet = useTonWallet();
  const rawAddress = useTonAddress(false);

  // Tick every second for deposit deadline countdown (updated_at + 1h)
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (deal?.status !== 'waiting_escrow_deposit') return;
    const interval = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(interval);
  }, [deal?.status]);
  const depositDeadlineMs = deal?.updated_at
    ? new Date(deal.updated_at).getTime() + 60 * 60 * 1000
    : 0;
  const depositTimeLeftMs = Math.max(0, depositDeadlineMs - now);
  const depositDeadlinePassed = deal?.status === 'waiting_escrow_deposit' && depositDeadlineMs > 0 && depositTimeLeftMs === 0;
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
    draftEditorsSyncedRef.current = false; // reset so this deal's first load can sync draft fields
    let isMounted = true;
    const fetchDeal = () => {
      api<Deal>(`/api/v1/market/deals/${id}`)
        .then((res) => {
          if (!isMounted) return;
          if (res.ok && res.data) {
            setDeal(res.data);
            setError(null);
            // When status is draft, sync draft editors only on first load; polling must not overwrite user input.
            const d = res.data.details as Record<string, unknown>;
            if (res.data.status !== 'draft') {
              draftEditorsSyncedRef.current = false; // allow sync again if deal goes back to draft
              setDraftMessage(getDealMessage(d));
              setDraftPostedAt(isoToDatetimeLocal(getDealPostedAt(d)));
            } else if (!draftEditorsSyncedRef.current) {
              draftEditorsSyncedRef.current = true;
              setDraftMessage(getDealMessage(d));
              setDraftPostedAt(isoToDatetimeLocal(getDealPostedAt(d)));
            }
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

  const handleJumpIntoChat = async () => {
    const authRes = await auth();
    if (!authRes.ok || !authRes.data) {
      alert('Open from Telegram to use deal chat.');
      return;
    }
    setAuthToken(authRes.data);
    setChatLinkLoading(true);
    try {
      const res = await api<{ chat_link: string }>(`/api/v1/market/deals/${id}/chat-link`, { method: 'POST' });
      if (res.ok && res.data?.chat_link) {
        openTelegramLink(res.data.chat_link);
      } else {
        alert(res.error_code || 'Could not open chat');
      }
    } finally {
      setChatLinkLoading(false);
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
        <div className="w-full space-y-4">
          {/* Wallet */}
          {(isLessor || isLessee) && (
            <div className="flex w-full max-w-md flex-col gap-1">
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
            </div>
          )}

          <DealStatusRoadmap currentStatus={deal.status} />

          {/* Row: Jump into chat + View stats (same height; each takes full width if alone) */}
          <div className="flex flex-wrap gap-2">
            {deal.status !== 'rejected' && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="inline-flex h-10 min-w-0 flex-1 items-center justify-center gap-2"
                onClick={handleJumpIntoChat}
                disabled={chatLinkLoading}
              >
                <MessageCircle className="h-4 w-4 shrink-0" aria-hidden />
                {chatLinkLoading ? 'Opening…' : 'Jump into chat'}
              </Button>
            )}
            {deal.status === 'draft' && isLessee && (deal.channel_id ?? listing?.channel_id) != null && (
              <Link
                href={`/profile/channels/${deal.channel_id ?? listing?.channel_id}`}
                className="inline-flex h-10 min-w-0 flex-1 items-center justify-center gap-2 rounded-lg border border-border bg-card px-4 py-2 text-sm font-medium hover:bg-accent"
              >
                <BarChart3 size={18} />
                View stats
              </Link>
            )}
          </div>

          {/* No time left for deposit - both sides */}
          {deal.status === 'waiting_escrow_deposit' && depositDeadlinePassed && (isLessor || isLessee) && (
            <p className="text-sm text-destructive font-medium">
              No time left for deposit, deal will be expired soon.
            </p>
          )}

          {/* Deposit to escrow - lessee only, only when time left; timer on the right */}
          {deal.status === 'waiting_escrow_deposit' && !depositDeadlinePassed && isLessee && deal.escrow_address != null && (
            <div className="rounded-md border border-border bg-muted/30 p-3 text-sm">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="space-y-2 min-w-0 flex-1">
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
                      className="w-full"
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
                <div className="shrink-0 text-right tabular-nums text-muted-foreground">
                  <p className="text-xs font-medium">Time left</p>
                  <p className="text-lg font-semibold text-foreground">{formatTimeLeftMs(depositTimeLeftMs)}</p>
                </div>
              </div>
            </div>
          )}

          {/* Type, Duration, Price, Listing, Post date, Post text */}
          <p className="text-sm">
            <strong>Type:</strong> Post
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
          {getDealPostedAt(deal.details as Record<string, unknown>) && (
            <p className="text-sm">
              <strong>Post date:</strong> {formatPostedAt(getDealPostedAt(deal.details as Record<string, unknown>))}
            </p>
          )}
          {getDealMessage(deal.details as Record<string, unknown>) && (
            <div>
              <p className="text-sm font-medium">Post text</p>
              <p className="mt-1 whitespace-pre-wrap rounded-md border border-border bg-muted/50 p-2 text-sm">
                {getDealMessage(deal.details as Record<string, unknown>)}
              </p>
            </div>
          )}

          {/* Handshake: draft (tap to sign) and approved until waiting_escrow_deposit */}
          {(deal.status === 'draft' || deal.status === 'approved') && (isLessor || isLessee) && (
            <HandshakeDealSign
              lessorSigned={lessorSigned}
              lesseeSigned={lesseeSigned}
              isLessor={!!isLessor}
              bothPayoutsSet={!!bothPayoutsSet}
              myWalletConnected={!!wallet}
              canSignNow={!!canSignNow}
              signing={!!signing}
              onSignDeal={handleSignDeal}
            />
          )}

          {deal.status === 'draft' && listing && (() => {
            const priceRows = parseListingPrices(listing.prices);
            if (priceRows.length === 0) return null;
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
                    className="mt-1 w-full min-w-[18rem] max-w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                  />
                  {draftPostedAtError && (
                    <p className="mt-1 text-xs text-destructive">{draftPostedAtError}</p>
                  )}
                </div>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    className="min-w-0 flex-1"
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
                    className="min-w-0 flex-1 text-destructive hover:bg-destructive/10 hover:text-destructive"
                    onClick={handleRejectDeal}
                    disabled={signing || rejecting || draftSaving}
                  >
                    {rejecting ? 'Rejecting…' : 'Reject deal'}
                  </Button>
                </div>
              </div>
            );
          })()}
          <p className="text-xs text-muted-foreground">
            Updated {new Date(deal.updated_at).toLocaleString()}
          </p>
        </div>
      </div>
          </div>
        ) : null}
      </div>
      <LoadingScreen show={loading} />
    </>
  );
}
