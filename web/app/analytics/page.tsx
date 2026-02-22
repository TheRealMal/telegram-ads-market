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
} from 'recharts';

type AuthStatus = 'loading' | 'unauthenticated' | 'unauthorized' | 'admin';

const ADMIN_ROLE = 'admin';

function formatNanotonToTon(nanoton: number): string {
  if (nanoton == null || Number.isNaN(nanoton)) return '—';
  const ton = nanoton / 1e9;
  return ton >= 1 ? `${ton.toFixed(1)} TON` : `${(ton * 1000).toFixed(2)} mTON`;
}

function SnapshotCards({ snapshot }: { snapshot: AnalyticsSnapshot | null }) {
  if (!snapshot) {
    return (
      <p className="rounded-lg border border-border bg-card p-4 text-center text-muted-foreground">
        No snapshot data yet. Snapshots are collected hourly.
      </p>
    );
  }
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
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
          {formatNanotonToTon(snapshot.commission_earned_nanoton)}
        </p>
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
