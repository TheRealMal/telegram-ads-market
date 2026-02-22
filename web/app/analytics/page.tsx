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
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
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

const PIE_COLORS = [
  'var(--primary)',
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
];

const RADIAN = Math.PI / 180;
function renderPieLabelInside(
  props: { cx: number; cy: number; midAngle: number; innerRadius: number; outerRadius: number; name: string; value: number; percent: number },
  valueFormat?: (v: number) => string
) {
  const { cx, cy, midAngle, innerRadius, outerRadius, name, value, percent } = props;
  if (percent < 0.06) return null;
  const r = (innerRadius + outerRadius) / 2;
  const x = cx + r * Math.cos(-midAngle * RADIAN);
  const y = cy + r * Math.sin(-midAngle * RADIAN);
  const text = valueFormat ? `${name}: ${valueFormat(value)}` : `${name}: ${value}`;
  return (
    <text x={x} y={y} fill="white" textAnchor="middle" dominantBaseline="central" fontSize={11} fontWeight={500}>
      {text}
    </text>
  );
}

function SnapshotCards({ snapshot }: { snapshot: AnalyticsSnapshot | null }) {
  if (!snapshot) {
    return (
      <p className="rounded-lg border border-border bg-card p-4 text-center text-muted-foreground">
        No snapshot data yet. Snapshots are collected hourly.
      </p>
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
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Listings
          </p>
          <p className="mt-1 text-2xl font-semibold">{snapshot.listings_count}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Deals
          </p>
          <p className="mt-1 text-2xl font-semibold">{snapshot.deals_count}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Users
          </p>
          <p className="mt-1 text-2xl font-semibold">{snapshot.users_count}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Commission earned
          </p>
          <p className="mt-1 text-2xl font-semibold">
            {formatTon(snapshot.commission_earned_ton)}
          </p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Avg listings / user
          </p>
          <p className="mt-1 text-2xl font-semibold">
            {snapshot.avg_listings_per_user != null && !Number.isNaN(snapshot.avg_listings_per_user)
              ? snapshot.avg_listings_per_user.toFixed(2)
              : '—'}
          </p>
        </div>
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

function DealsByStatusPie({ snapshot }: { snapshot: AnalyticsSnapshot | null }) {
  if (!snapshot?.deals_by_status || Object.keys(snapshot.deals_by_status).length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card p-6">
        <h2 className="mb-4 text-lg font-semibold">Deals by status</h2>
        <p className="text-center text-sm text-muted-foreground">No deal status data.</p>
      </div>
    );
  }
  const data = mapToPieData(snapshot.deals_by_status);
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h2 className="mb-4 text-lg font-semibold">Deals by status</h2>
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
              label={(p) => renderPieLabelInside(p)}
            >
              {data.map((_, i) => (
                <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--card)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius)',
                color: 'var(--card-foreground)',
              }}
              itemStyle={{ color: 'var(--card-foreground)' }}
              labelStyle={{ color: 'var(--card-foreground)' }}
              formatter={(value: number) => [value, 'Deals']}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

function DealAmountsByStatusPie({ snapshot }: { snapshot: AnalyticsSnapshot | null }) {
  if (!snapshot?.deal_amounts_by_status_ton || Object.keys(snapshot.deal_amounts_by_status_ton).length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card p-6">
        <h2 className="mb-4 text-lg font-semibold">Deal amounts by status (TON)</h2>
        <p className="text-center text-sm text-muted-foreground">No amount data.</p>
      </div>
    );
  }
  const data = mapToPieData(snapshot.deal_amounts_by_status_ton);
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h2 className="mb-4 text-lg font-semibold">Deal amounts by status (TON)</h2>
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
              label={(p) => renderPieLabelInside(p, (v) => `${v.toFixed(2)} TON`)}
            >
              {data.map((_, i) => (
                <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--card)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius)',
                color: 'var(--card-foreground)',
              }}
              itemStyle={{ color: 'var(--card-foreground)' }}
              labelStyle={{ color: 'var(--card-foreground)' }}
              formatter={(value: number) => [`${value.toFixed(2)} TON`, 'Amount']}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

function HistoryCharts({
  history,
  period,
  onPeriodChange,
}: {
  history: AnalyticsHistoryResponse | null;
  period: string;
  onPeriodChange: (p: string) => void;
}) {
  if (!history || !history.timestamps.length) {
    return (
      <div className="rounded-lg border border-border bg-card p-6">
        <p className="text-center text-muted-foreground">
          No history data for the selected period.
        </p>
      </div>
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

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-lg font-semibold">History</h2>
        <div className="flex gap-2">
          {(['week', 'month', 'year'] as const).map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => onPeriodChange(p)}
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                period === p
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-muted/80'
              }`}
            >
              {p.charAt(0).toUpperCase() + p.slice(1)}
            </button>
          ))}
        </div>
      </div>
      <div className="h-72 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
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
            <Legend />
            <Line type="monotone" dataKey="listings" stroke="var(--primary)" name="Listings" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="deals" stroke="#22c55e" name="Deals" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="users" stroke="#3b82f6" name="Users" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="commission_ton" stroke="#f59e0b" name="Commission (TON)" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="avg_listings" stroke="#8b5cf6" name="Avg listings/user" strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export default function AnalyticsPage() {
  const [authStatus, setAuthStatus] = useState<AuthStatus>('loading');
  const [latest, setLatest] = useState<LatestSnapshotResponse['snapshot']>(null);
  const [history, setHistory] = useState<AnalyticsHistoryResponse | null>(null);
  const [period, setPeriod] = useState<'week' | 'month' | 'year'>('week');
  const [dataLoading, setDataLoading] = useState(false);

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
        className={`min-h-screen pb-8 ${showContent ? 'opacity-100' : 'opacity-0'}`}
      >
        <PageTopSpacer />
        <div className="mx-auto max-w-4xl px-4 py-6">
          <h1 className="mb-6 text-2xl font-bold">Analytics Dashboard</h1>

          {authStatus === 'unauthenticated' && (
            <div className="rounded-lg border border-border bg-card p-6 text-center">
              <p className="text-muted-foreground">
                Please open this app from Telegram and sign in to continue.
              </p>
            </div>
          )}

          {authStatus === 'unauthorized' && (
            <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6 text-center">
              <p className="font-medium text-destructive">Unauthorized</p>
              <p className="mt-2 text-sm text-muted-foreground">
                Only administrators can view the analytics dashboard.
              </p>
            </div>
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
