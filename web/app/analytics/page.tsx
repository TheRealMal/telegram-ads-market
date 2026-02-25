'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  api,
  ensureValidToken,
  getRoleFromToken,
} from '@/lib/api';
import type {
  AnalyticsHistoryResponse,
  AnalyticsSnapshot,
  LatestSnapshotResponse,
} from '@/types';
import { getDealStatusLabel } from '@/types';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { LoadingScreen } from '@/components/LoadingScreen';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts';

type AuthStatus = 'loading' | 'unauthenticated' | 'unauthorized' | 'admin';

const ADMIN_ROLE = 'admin';

function formatTon(ton: number): string {
  if (ton == null || Number.isNaN(ton)) return '—';
  return ton >= 1 ? `${ton.toFixed(1)} TON` : `${(ton * 1000).toFixed(2)} mTON`;
}

/** No white/black; distinct colors so slices and text stay visible. */
const PIE_COLORS = [
  '#22c55e',
  '#3b82f6',
  '#f59e0b',
  '#ef4444',
  '#8b5cf6',
  '#ec4899',
  '#06b6d4',
  '#84cc16',
  '#f97316',
  '#6366f1',
  '#14b8a6',
  '#a855f7',
];

const HISTORY_COLORS = [
  '#22c55e',
  '#3b82f6',
  '#f59e0b',
  '#8b5cf6',
  '#ec4899',
];

const RADIAN = Math.PI / 180;
function renderPieLabelInside(
  props: { cx: number; cy: number; midAngle: number; innerRadius: number; outerRadius: number; name?: string; percent: number; fill?: string; payload?: { name?: string; fill?: string } }
) {
  const { cx, cy, midAngle, innerRadius, outerRadius, percent, payload } = props;
  const name = props.name ?? payload?.name ?? '';
  if (!name || percent < 0.04) return null;
  const r = (innerRadius + outerRadius) / 2;
  const x = cx + r * Math.cos(-midAngle * RADIAN);
  const y = cy + r * Math.sin(-midAngle * RADIAN);
  return (
    <text
      x={x}
      y={y}
      fill="white"
      stroke="rgba(0,0,0,0.4)"
      strokeWidth={1.5}
      textAnchor="middle"
      dominantBaseline="central"
      fontSize={11}
      fontWeight={500}
    >
      {name}
    </text>
  );
}

function SnapshotCards({ snapshot }: { snapshot: AnalyticsSnapshot | null }) {
  if (!snapshot) {
    return (
      <Card>
        <CardContent className="py-6">
          <p className="text-center text-muted-foreground">
            No snapshot data yet. Snapshots are collected hourly.
          </p>
        </CardContent>
      </Card>
    );
  }
  const recordedAt = snapshot.recorded_at
    ? new Date(snapshot.recorded_at).toLocaleString(undefined, {
        dateStyle: 'short',
        timeStyle: 'short',
      })
    : '—';
  return (
    <div className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        <Card>
          <CardContent className="p-4">
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Listings
            </p>
            <p className="mt-1 text-2xl font-semibold">{snapshot.listings_count}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Deals
            </p>
            <p className="mt-1 text-2xl font-semibold">{snapshot.deals_count}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Users
            </p>
            <p className="mt-1 text-2xl font-semibold">{snapshot.users_count}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Commission earned
            </p>
            <p className="mt-1 text-2xl font-semibold">
              {formatTon(snapshot.commission_earned_ton)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Avg listings / user
            </p>
            <p className="mt-1 text-2xl font-semibold">
              {snapshot.avg_listings_per_user != null && !Number.isNaN(snapshot.avg_listings_per_user)
                ? snapshot.avg_listings_per_user.toFixed(2)
                : '—'}
            </p>
          </CardContent>
        </Card>
      </div>
      <p className="text-xs text-muted-foreground">Last snapshot: {recordedAt}</p>
    </div>
  );
}

function mapToPieData(data: Record<string, number>): { name: string; value: number }[] {
  return Object.entries(data)
    .filter(([, v]) => v != null && Number(v) > 0)
    .map(([status, value]) => ({
      name: getDealStatusLabel(status),
      value: Number(value),
    }))
    .sort((a, b) => b.value - a.value);
}

function PieTooltipContent({
  payload,
  valueLabel,
  valueFormat,
}: {
  payload?: Array<{ name?: unknown; value?: unknown; payload?: { fill?: string }; color?: string }>;
  valueLabel: string;
  valueFormat?: (v: number) => string;
}) {
  if (!payload?.length) return null;
  const item = payload[0];
  const color = item.payload?.fill ?? item.color ?? 'var(--foreground)';
  const value = Number(item.value ?? 0);
  const valueStr = valueFormat ? valueFormat(value) : String(value);
  const name = item.name != null ? String(item.name) : '';
  return (
    <div
      className="rounded-md border border-border bg-card px-3 py-2 text-sm shadow-md"
      style={{ borderColor: 'var(--border)' }}
    >
      <div className="font-medium" style={{ color }}>
        {name}
      </div>
      <div className="mt-0.5 text-muted-foreground">
        {valueLabel}: {valueStr}
      </div>
    </div>
  );
}

function DealsByStatusPie({ snapshot }: { snapshot: AnalyticsSnapshot | null }) {
  if (!snapshot?.deals_by_status || Object.keys(snapshot.deals_by_status).length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Deals by status</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-center text-sm text-muted-foreground">No deal status data.</p>
        </CardContent>
      </Card>
    );
  }
  const rawData = mapToPieData(snapshot.deals_by_status);
  const data = rawData.map((d, i) => ({ ...d, fill: PIE_COLORS[i % PIE_COLORS.length] }));
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Deals by status</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-72 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={data}
                dataKey="value"
                nameKey="name"
                cx="50%"
                cy="50%"
                outerRadius="80%"
                labelLine={false}
                label={(p) => renderPieLabelInside(p)}
              >
                {data.map((_, i) => (
                  <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
                ))}
              </Pie>
              <Tooltip
                content={({ payload }) => (
                  <PieTooltipContent payload={payload} valueLabel="Deals" />
                )}
              />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}

function DealAmountsByStatusPie({ snapshot }: { snapshot: AnalyticsSnapshot | null }) {
  if (!snapshot?.deal_amounts_by_status_ton || Object.keys(snapshot.deal_amounts_by_status_ton).length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Deal amounts by status (TON)</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-center text-sm text-muted-foreground">No amount data.</p>
        </CardContent>
      </Card>
    );
  }
  const rawData = mapToPieData(snapshot.deal_amounts_by_status_ton);
  const data = rawData.map((d, i) => ({ ...d, fill: PIE_COLORS[i % PIE_COLORS.length] }));
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Deal amounts by status (TON)</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-72 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={data}
                dataKey="value"
                nameKey="name"
                cx="50%"
                cy="50%"
                outerRadius="80%"
                labelLine={false}
                label={(p) => renderPieLabelInside(p)}
              >
                {data.map((_, i) => (
                  <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
                ))}
              </Pie>
              <Tooltip
                content={({ payload }) => (
                  <PieTooltipContent
                    payload={payload}
                    valueLabel="Amount"
                    valueFormat={(v) => `${v.toFixed(2)} TON`}
                  />
                )}
              />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}

const HISTORY_SERIES: { dataKey: string; name: string }[] = [
  { dataKey: 'listings', name: 'Listings' },
  { dataKey: 'deals', name: 'Deals' },
  { dataKey: 'users', name: 'Users' },
  { dataKey: 'commission_ton', name: 'Commission (TON)' },
  { dataKey: 'avg_listings', name: 'Avg listings/user' },
];

function HistoryCharts({
  history,
  period,
  onPeriodChange,
  hiddenSeries,
  onToggleSeries,
}: {
  history: AnalyticsHistoryResponse | null;
  period: string;
  onPeriodChange: (p: string) => void;
  hiddenSeries: Set<string>;
  onToggleSeries: (dataKey: string) => void;
}) {
  if (!history || !history.timestamps.length) {
    return (
      <Card>
        <CardContent className="py-6">
          <p className="text-center text-muted-foreground">
            No history data for the selected period.
          </p>
        </CardContent>
      </Card>
    );
  }

  const chartData = history.timestamps.map((ts, i) => ({
    time: new Date(ts * 1000).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
    }),
    listings: history.listings_count[i] ?? 0,
    deals: history.deals_count[i] ?? 0,
    users: history.users_count[i] ?? 0,
    commission_ton: history.commission_earned_ton[i] ?? 0,
    avg_listings: history.avg_listings_per_user[i] ?? 0,
  }));

  const isVisible = (dataKey: string) => !hiddenSeries.has(dataKey);

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-base">History</CardTitle>
          <div className="flex gap-1">
            {(['week', 'month', 'year'] as const).map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => onPeriodChange(p)}
                className={`rounded-full border px-3 py-1.5 text-sm font-medium transition-colors ${
                  period === p
                    ? 'border-primary bg-primary text-primary-foreground'
                    : 'border-border bg-transparent text-muted-foreground hover:bg-muted/80'
                }`}
              >
                {p.charAt(0).toUpperCase() + p.slice(1)}
              </button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent className="px-3 pb-0 pt-0">
        <div className="h-72 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis dataKey="time" className="text-xs" tick={{ fill: 'currentColor' }} />
              <YAxis className="text-xs" tick={{ fill: 'currentColor' }} />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--card)',
                  border: '1px solid var(--border)',
                  borderRadius: 'var(--radius)',
                }}
                labelStyle={{ color: 'var(--card-foreground)' }}
              />
              {HISTORY_SERIES.filter((s) => isVisible(s.dataKey)).map((s, i) => (
                <Area
                  key={s.dataKey}
                  type="monotone"
                  dataKey={s.dataKey}
                  name={s.name}
                  stroke={HISTORY_COLORS[HISTORY_SERIES.findIndex((x) => x.dataKey === s.dataKey)]}
                  fill={HISTORY_COLORS[HISTORY_SERIES.findIndex((x) => x.dataKey === s.dataKey)]}
                  fillOpacity={0.35}
                  strokeWidth={2}
                  dot={false}
                  isAnimationActive={true}
                />
              ))}
            </AreaChart>
          </ResponsiveContainer>
        </div>
        <div className="flex flex-wrap justify-center gap-2 pt-2 pb-1">
          {HISTORY_SERIES.map((s, i) => {
            const hidden = hiddenSeries.has(s.dataKey);
            const color = HISTORY_COLORS[i];
            return (
              <button
                key={s.dataKey}
                type="button"
                onClick={() => onToggleSeries(s.dataKey)}
                className="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition-opacity hover:opacity-90"
                style={{
                  backgroundColor: hidden ? 'transparent' : color,
                  borderColor: color,
                  color: hidden ? color : 'white',
                  transition: 'background-color 0.2s ease-out, color 0.2s ease-out',
                }}
              >
                <span
                  className="flex h-4 w-4 shrink-0 items-center justify-center text-[10px] font-bold"
                  style={{ opacity: hidden ? 0 : 1 }}
                >
                  ✓
                </span>
                <span>{s.name}</span>
              </button>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

export default function AnalyticsPage() {
  const [authStatus, setAuthStatus] = useState<AuthStatus>('loading');
  const [latest, setLatest] = useState<LatestSnapshotResponse['snapshot']>(null);
  const [history, setHistory] = useState<AnalyticsHistoryResponse | null>(null);
  const [period, setPeriod] = useState<'week' | 'month' | 'year'>('week');
  const [dataLoading, setDataLoading] = useState(false);
  const [hiddenHistorySeries, setHiddenHistorySeries] = useState<Set<string>>(new Set());

  const toggleHistorySeries = useCallback((dataKey: string) => {
    setHiddenHistorySeries((prev) => {
      const next = new Set(prev);
      if (next.has(dataKey)) next.delete(dataKey);
      else next.add(dataKey);
      return next;
    });
  }, []);

  const fetchData = useCallback(async () => {
    setDataLoading(true);
    try {
      const [latestRes, historyRes] = await Promise.all([
        api<LatestSnapshotResponse>('/api/v1/analytics/snapshot/latest'),
        api<AnalyticsHistoryResponse>(
          `/api/v1/analytics/snapshot/history?period=${period}`
        ),
      ]);
      if (latestRes.ok && latestRes.data) setLatest(latestRes.data.snapshot);
      if (historyRes.ok && historyRes.data) setHistory(historyRes.data);
    } finally {
      setDataLoading(false);
    }
  }, [period]);

  useEffect(() => {
    ensureValidToken().then((token) => {
      if (!token) {
        setAuthStatus('unauthenticated');
        return;
      }
      const role = getRoleFromToken(token);
      if (role !== ADMIN_ROLE) {
        setAuthStatus('unauthorized');
        return;
      }
      setAuthStatus('admin');
    });
  }, []);

  useEffect(() => {
    if (authStatus !== 'admin') return;
    fetchData();
  }, [authStatus, fetchData]);

  const showContent = authStatus !== 'loading';
  const showDashboard = authStatus === 'admin';

  return (
    <>
      <div
        className={`page-with-nav ${showContent ? 'opacity-100' : 'opacity-0'}`}
      >
        <PageTopSpacer />
        <div className="mx-auto max-w-4xl px-4 py-4">
          <h1 className="mb-6 text-2xl font-bold">Analytics Dashboard</h1>

          {authStatus === 'unauthenticated' && (
            <Card>
              <CardContent className="py-6 text-center">
                <p className="text-muted-foreground">
                  Please open this app from Telegram and sign in to continue.
                </p>
              </CardContent>
            </Card>
          )}

          {authStatus === 'unauthorized' && (
            <Card className="border-destructive/50 bg-destructive/10">
              <CardContent className="py-6 text-center">
                <p className="font-medium text-destructive">Unauthorized</p>
                <p className="mt-2 text-sm text-muted-foreground">
                  Only administrators can view the analytics dashboard.
                </p>
              </CardContent>
            </Card>
          )}

          {showDashboard && (
            <div className="space-y-6">
              <SnapshotCards snapshot={latest} />
              <div className="grid gap-6 lg:grid-cols-2">
                <DealsByStatusPie snapshot={latest} />
                <DealAmountsByStatusPie snapshot={latest} />
              </div>
              <HistoryCharts
                history={history}
                period={period}
                onPeriodChange={(p) => setPeriod(p as 'week' | 'month' | 'year')}
                hiddenSeries={hiddenHistorySeries}
                onToggleSeries={toggleHistorySeries}
              />
              {dataLoading && (
                <p className="text-center text-sm text-muted-foreground">
                  Updating data…
                </p>
              )}
            </div>
          )}
        </div>
      </div>
      <LoadingScreen show={!showContent} />
    </>
  );
}
