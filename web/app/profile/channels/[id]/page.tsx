'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useTelegramBackButton } from '@/lib/telegram';
import type { Channel } from '@/types';
import type { ChannelStatsResponse } from '@/types/channelStats';
import {
  parseGraphData,
  getGraphChartConfig,
  getGraphTitle,
  STATS_GRAPH_ORDER,
  getStatsDelta,
} from '@/types/channelStats';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { PageTopSpacer } from '@/components/PageTopSpacer';

export default function ChannelStatsPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;
  useTelegramBackButton(() => router.push('/profile'));
  const [channel, setChannel] = useState<Channel | null>(null);
  const [stats, setStats] = useState<ChannelStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    const numId = parseInt(id, 10);
    if (Number.isNaN(numId)) {
      setError('Invalid channel ID');
      setLoading(false);
      return;
    }
    // We don't have a public GET channel by id; we have my-channels. So we fetch stats and if 200 we show stats (channel title can come from stats or we could add channel to response). For now we just fetch stats - the backend returns 403 if no access. If we get 200 we have stats. We don't have channel name here unless we add it to the stats response or fetch my-channels and find by id. Let me fetch my-channels and find the channel for the title, and fetch stats.
    Promise.all([
      api<Channel[]>('/api/v1/market/my-channels'),
      api<ChannelStatsResponse>(`/api/v1/market/channels/${numId}/stats`),
    ])
      .then(([chRes, statsRes]) => {
        if (chRes.ok && chRes.data) {
          const ch = chRes.data.find((c) => c.id === numId);
          if (ch) setChannel(ch);
        }
        if (statsRes.ok && statsRes.data) {
          setStats(statsRes.data);
        } else {
          setError(statsRes.error_code || 'Cannot view stats for this channel');
        }
      })
      .catch(() => setError('Request failed'))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  if (error && !stats) {
    return (
      <div className="min-h-screen pb-20">
        <PageTopSpacer />
        <div className="mx-auto max-w-2xl px-4 py-8">
          <p className="text-destructive">{error}</p>
        </div>
      </div>
    );
  }

  const followers = stats?.Followers;
  const viewsPerPost = stats?.ViewsPerPost;
  const sharesPerPost = stats?.SharesPerPost;
  const reactionsPerPost = stats?.ReactionsPerPost;
  const viewsPerStory = stats?.ViewsPerStory;
  const sharesPerStory = stats?.SharesPerStory;
  const reactionsPerStory = stats?.ReactionsPerStory;
  const enabledNotifications = stats?.EnabledNotifications;
  const recentPosts = stats?.RecentPostsInteractions ?? [];
  const period = stats?.Period;
  const periodLabel =
    period?.MinDate != null && period?.MaxDate != null
      ? `${new Date(period.MinDate * 1000).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })} – ${new Date(period.MaxDate * 1000).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}`
      : null;

  /** Single stat row: label, current value, optional red/green delta (%). Hide delta when value and delta are both 0. */
  function StatWithDelta({
    label,
    current,
    previous,
  }: {
    label: string;
    current: number | undefined;
    previous: number | undefined;
  }) {
    const c = current ?? 0;
    const { deltaPercent, showDelta } = getStatsDelta(current, previous);
    return (
      <div className="flex items-baseline justify-between gap-2 py-1">
        <span className="text-sm text-muted-foreground">{label}</span>
        <span className="flex items-baseline gap-1.5">
          <span className="font-semibold">{c}</span>
          {showDelta && (
            <span className={deltaPercent > 0 ? 'text-green-600' : deltaPercent < 0 ? 'text-red-600' : 'text-muted-foreground'}>
              {deltaPercent > 0 ? '+' : ''}{deltaPercent.toFixed(1)}%
            </span>
          )}
        </span>
      </div>
    );
  }

  const GREEN = '#22c55e';
  const RED = '#ef4444';
  const BLUE = '#3b82f6';
  const YELLOW = '#eab308';
  const BLACK = '#000000';
  const GRAY = '#6b7280';

  /** Colorful fallback palette for series with no specific color (no black/gray/white). Constant order by index. */
  const DEFAULT_COLORS = [
    '#22c55e', // green
    '#3b82f6', // blue
    '#eab308', // yellow
    '#8b5cf6', // violet
    '#ec4899', // pink
    '#06b6d4', // cyan
    '#f97316', // orange
    '#14b8a6', // teal
    '#a855f7', // purple
  ];

  function getSeriesColor(
    graphKey: string,
    index: number,
    colKey: string,
    colName: string
  ): string {
    const k = (colKey || colName || '').toLowerCase();
    const n = (colName || colKey || '').toLowerCase();
    const label = n || k;

    switch (graphKey) {
      case 'FollowersGraph':
        if (label.includes('join') || index === 0) return GREEN;
        if (label.includes('left') || index === 1) return RED;
        break;
      case 'InteractionsGraph':
        if (label.includes('view')) return GREEN;
        if (label.includes('share')) return BLUE;
        break;
      case 'MuteGraph':
        if (label.includes('mute') && !label.includes('un')) return RED;
        if (label.includes('unmute') || label.includes('un mute')) return GREEN;
        break;
      case 'ReactionsByEmotionGraph':
        if (label.includes('positive')) return GREEN;
        return YELLOW;
      case 'TopHoursGraph':
        return index === 0 ? BLACK : GRAY;
      default:
        break;
    }
    return DEFAULT_COLORS[index % DEFAULT_COLORS.length];
  }

  const graphEntries = (stats ? STATS_GRAPH_ORDER : [])
    .map((key) => {
      const g = (stats as Record<string, unknown>)[key] as { JSON?: { Data?: string }; Error?: string } | undefined;
      if (!g || g.Error) return null;
      const dataStr = g?.JSON?.Data;
      const data = parseGraphData(dataStr);
      const config = getGraphChartConfig(data);
      return config && config.rows.length > 0 ? { key, title: getGraphTitle(key), config } : null;
    })
    .filter((e): e is NonNullable<typeof e> => e != null);

  return (
    <div className="min-h-screen pb-20">
      <PageTopSpacer />
      <div className="mx-auto max-w-2xl space-y-4 px-4 py-5">
        {/* Overview: two columns */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Overview</CardTitle>
            {periodLabel && (
              <p className="text-sm text-muted-foreground">{periodLabel}</p>
            )}
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-6 gap-y-0">
              <div className="space-y-0">
                {followers != null && (
                  <StatWithDelta label="Followers" current={followers.Current} previous={followers.Previous} />
                )}
                {viewsPerPost != null && (
                  <StatWithDelta label="Views per post" current={viewsPerPost.Current} previous={viewsPerPost.Previous} />
                )}
                {sharesPerPost != null && (
                  <StatWithDelta label="Shares per post" current={sharesPerPost.Current} previous={sharesPerPost.Previous} />
                )}
                {reactionsPerPost != null && (
                  <StatWithDelta label="Reactions per post" current={reactionsPerPost.Current} previous={reactionsPerPost.Previous} />
                )}
              </div>
              <div className="space-y-0">
                {enabledNotifications != null && (
                  <div className="flex items-baseline justify-between gap-2 py-1">
                    <span className="text-sm text-muted-foreground">Enabled notifications</span>
                    <span className="font-semibold">
                      {enabledNotifications.Total
                        ? Math.round(((enabledNotifications.Part ?? 0) / enabledNotifications.Total) * 100)
                        : 0}%
                    </span>
                  </div>
                )}
                {viewsPerStory != null && (
                  <StatWithDelta label="Views per story" current={viewsPerStory.Current} previous={viewsPerStory.Previous} />
                )}
                {sharesPerStory != null && (
                  <StatWithDelta label="Shares per story" current={sharesPerStory.Current} previous={sharesPerStory.Previous} />
                )}
                {reactionsPerStory != null && (
                  <StatWithDelta label="Reactions per story" current={reactionsPerStory.Current} previous={reactionsPerStory.Previous} />
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Graphs in fixed order */}
        {graphEntries.map(({ key, title, config }) => {
          const { rows, yColumns, xLabel, yLabel, chartType } = config;
          const formatX = (ts: number) =>
            ts > 1e10
              ? new Date(ts).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
              : String(ts);
          const tooltipLabel = (ts: number) =>
            ts > 1e10 ? new Date(ts).toLocaleString() : String(ts);

          // 100% stacked area for Languages
          const isLanguages = key === 'LanguagesGraph';
          const chartRows = isLanguages
            ? rows.map((row) => {
                const sum = yColumns.reduce((s, col) => s + (Number(row[col.key]) || 0), 0);
                const out: Record<string, number> = { x: row.x };
                for (const col of yColumns) {
                  out[col.key] = sum > 0 ? (Number(row[col.key]) || 0) / sum * 100 : 0;
                }
                return out;
              })
            : rows;

          return (
            <Card key={key}>
              <CardHeader>
                <CardTitle className="text-base">{title}</CardTitle>
                {periodLabel && (
                  <p className="text-sm text-muted-foreground">{periodLabel}</p>
                )}
              </CardHeader>
              <CardContent className="px-3 pb-0 pt-0">
                <div className="h-72 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    {isLanguages ? (
                      <AreaChart data={chartRows} margin={{ top: 5, right: 5, left: 0, bottom: 25 }}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                        <XAxis
                          dataKey="x"
                          tickFormatter={formatX}
                          label={{ value: xLabel, position: 'insideBottom', offset: -5 }}
                          className="text-xs"
                        />
                        <YAxis
                          domain={[0, 100]}
                          tickFormatter={(v) => `${v}%`}
                          className="text-xs"
                          label={{ value: '%', angle: -90, position: 'insideLeft' }}
                        />
                        <Tooltip
                          labelFormatter={tooltipLabel}
                          formatter={(value: number, name: string) => [`${value.toFixed(1)}%`, name]}
                          contentStyle={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}
                        />
                        <Legend wrapperStyle={{ paddingTop: 8 }} />
                        {yColumns.map((col, i) => (
                          <Area
                            key={col.key}
                            type="monotone"
                            dataKey={col.key}
                            name={col.name}
                            stackId="1"
                            fill={getSeriesColor(key, i, col.key, col.name)}
                            stroke={getSeriesColor(key, i, col.key, col.name)}
                            strokeWidth={0}
                          />
                        ))}
                      </AreaChart>
                    ) : chartType === 'bar' ? (
                      <BarChart data={chartRows} margin={{ top: 5, right: 5, left: 0, bottom: 25 }}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                        <XAxis
                          dataKey="x"
                          tickFormatter={formatX}
                          label={{ value: xLabel, position: 'insideBottom', offset: -5 }}
                          className="text-xs"
                        />
                        <YAxis
                          className="text-xs"
                          label={{ value: yLabel, angle: -90, position: 'insideLeft' }}
                        />
                        <Tooltip
                          labelFormatter={tooltipLabel}
                          contentStyle={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}
                        />
                        <Legend wrapperStyle={{ paddingTop: 8 }} />
                        {yColumns.map((col, i) => (
                          <Bar
                            key={col.key}
                            dataKey={col.key}
                            name={col.name}
                            stackId="a"
                            fill={getSeriesColor(key, i, col.key, col.name)}
                          />
                        ))}
                      </BarChart>
                    ) : (
                      <LineChart data={chartRows} margin={{ top: 5, right: 5, left: 0, bottom: 25 }}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                        <XAxis
                          dataKey="x"
                          tickFormatter={formatX}
                          label={{ value: xLabel, position: 'insideBottom', offset: -5 }}
                          className="text-xs"
                        />
                        <YAxis
                          className="text-xs"
                          label={{ value: yLabel, angle: -90, position: 'insideLeft' }}
                        />
                        <Tooltip
                          labelFormatter={tooltipLabel}
                          contentStyle={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}
                        />
                        <Legend wrapperStyle={{ paddingTop: 8 }} />
                        {yColumns.map((col, i) => (
                          <Line
                            key={col.key}
                            type={col.type === 'step' ? 'step' : 'monotone'}
                            dataKey={col.key}
                            name={col.name}
                            stroke={getSeriesColor(key, i, col.key, col.name)}
                            strokeWidth={2}
                            dot={false}
                          />
                        ))}
                      </LineChart>
                    )}
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>
          );
        })}

        {/* Recent posts */}
        {recentPosts.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Recent posts</CardTitle>
              {periodLabel && (
                <p className="text-sm text-muted-foreground">{periodLabel}</p>
              )}
            </CardHeader>
            <CardContent>
              <ul className="space-y-2">
                {recentPosts.map((post, i) => {
                const msgId = post.MsgID ?? i + 1;
                const messageUrl = `https://t.me/c/${id}/${msgId}`;
                return (
                  <li
                    key={i}
                    className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2 text-sm"
                  >
                    <a
                      href={messageUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary underline hover:no-underline"
                    >
                      Message #{msgId}
                    </a>
                    <span>
                      {post.Views ?? 0} views · {post.Reactions ?? 0} reactions · {post.Forwards ?? 0} forwards
                    </span>
                  </li>
                );
              })}
              </ul>
            </CardContent>
          </Card>
        )}

        {!stats && (
          <p className="py-4 text-center text-sm text-muted-foreground">No stats data.</p>
        )}
      </div>
    </div>
  );
}
