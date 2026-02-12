/** Channel stats response shape (Telegram broadcast stats, see test.json). */

export interface StatsPeriod {
  MinDate?: number;
  MaxDate?: number;
}

export interface StatsCurrentPrevious {
  Current?: number;
  Previous?: number;
}

export interface StatsGraphData {
  columns?: [string, ...(number | string)[]][];
  types?: Record<string, string>;
  names?: Record<string, string>;
}

export interface StatsGraphJson {
  Data?: string; // JSON string of StatsGraphData
  Error?: string;
}

export interface RecentPostInteraction {
  MsgID?: number;
  Views?: number;
  Forwards?: number;
  Reactions?: number;
}

export interface ChannelStatsResponse {
  Period?: StatsPeriod;
  Followers?: StatsCurrentPrevious;
  ViewsPerPost?: StatsCurrentPrevious;
  SharesPerPost?: StatsCurrentPrevious;
  ReactionsPerPost?: StatsCurrentPrevious;
  ViewsPerStory?: StatsCurrentPrevious;
  SharesPerStory?: StatsCurrentPrevious;
  ReactionsPerStory?: StatsCurrentPrevious;
  ReactionsByEmotionGraph?: StatsGraphJson;
  EnabledNotifications?: { Part?: number; Total?: number };
  RecentPostsInteractions?: RecentPostInteraction[];
  [key: string]: unknown;
}

export function parseGraphData(jsonStr: string | undefined): StatsGraphData | null {
  if (!jsonStr) return null;
  try {
    return JSON.parse(jsonStr) as StatsGraphData;
  } catch {
    return null;
  }
}

const plottableTypes = ['line', 'step', 'bar'];

/** Extract x and first y column for a simple line/step/bar chart. */
export function getGraphSeries(data: StatsGraphData | null): { x: number; value: number }[] {
  if (!data?.columns?.length) return [];
  const cols = data.columns;
  const xCol = cols.find((c) => c[0] === 'x' || (data.types && data.types[c[0] as string] === 'x'));
  const yCol = cols.find((c) =>
    c[0] !== 'x' && plottableTypes.includes(data.types?.[c[0] as string] ?? '')
  );
  if (!xCol || !yCol || xCol.length !== yCol.length) return [];
  return xCol.slice(1).map((x, i) => ({ x: Number(x), value: Number(yCol[i + 1]) ?? 0 }));
}

export interface GraphSeriesColumn {
  key: string;
  name: string;
  type: 'line' | 'step' | 'bar';
}

export interface GraphChartConfig {
  rows: Record<string, number>[];
  yColumns: GraphSeriesColumn[];
  xLabel: string;
  yLabel: string;
  chartType: 'line' | 'bar';
}

/** Build multi-series chart config: rows with x + all y columns, axis labels, and series metadata. */
export function getGraphChartConfig(data: StatsGraphData | null): GraphChartConfig | null {
  if (!data?.columns?.length) return null;
  const cols = data.columns;
  const types = data.types ?? {};
  const names = data.names ?? {};
  const xCol = cols.find((c) => c[0] === 'x' || types[c[0] as string] === 'x');
  if (!xCol) return null;
  const yCols = cols.filter(
    (c) => c[0] !== 'x' && plottableTypes.includes(types[c[0] as string] ?? '')
  );
  if (yCols.length === 0) return null;
  const n = xCol.length - 1;
  const rows: Record<string, number>[] = [];
  for (let i = 0; i < n; i++) {
    const row: Record<string, number> = { x: Number(xCol[i + 1]) };
    for (const yCol of yCols) {
      const key = yCol[0] as string;
      row[key] = Number(yCol[i + 1]) ?? 0;
    }
    rows.push(row);
  }
  const yColumns: GraphSeriesColumn[] = yCols.map((yCol) => {
    const key = yCol[0] as string;
    return {
      key,
      name: (names[key] as string) || key,
      type: (types[key] as 'line' | 'step' | 'bar') || 'line',
    };
  });
  const firstX = rows[0]?.x;
  const xLabel =
    typeof firstX === 'number' && firstX > 1e10
      ? 'Date'
      : typeof firstX === 'number' && firstX >= 0 && firstX <= 23
        ? 'Hour (UTC)'
        : 'X';
  const hasBar = yColumns.some((s) => s.type === 'bar');
  return {
    rows,
    yColumns,
    xLabel,
    yLabel: 'Count',
    chartType: hasBar ? 'bar' : 'line',
  };
}

const graphTitleMap: Record<string, string> = {
  GrowthGraph: 'Growth',
  MuteGraph: 'Notifications',
  TopHoursGraph: 'View by hours (UTC)',
  FollowersGraph: 'Followers',
  LanguagesGraph: 'Languages',
  InteractionsGraph: 'Interactions',
  ViewsBySourceGraph: 'Views by source',
  NewFollowersBySourceGraph: 'New followers by source',
  ReactionsByEmotionGraph: 'Reactions',
};

/** Order in which stats graphs are displayed. */
export const STATS_GRAPH_ORDER: string[] = [
  'GrowthGraph',
  'FollowersGraph',
  'MuteGraph',
  'TopHoursGraph',
  'ViewsBySourceGraph',
  'NewFollowersBySourceGraph',
  'LanguagesGraph',
  'InteractionsGraph',
  'ReactionsByEmotionGraph',
];

export function getGraphTitle(key: string): string {
  return graphTitleMap[key] ?? (key.replace(/Graph$/, '').replace(/([A-Z])/g, ' $1').trim() || key);
}

/** Delta from previous to current. showDelta is false when both are 0 (display only "0"). */
export function getStatsDelta(
  current: number | undefined,
  previous: number | undefined
): { deltaPercent: number; showDelta: boolean } {
  const c = current ?? 0;
  const p = previous ?? 0;
  if (c === 0 && p === 0) return { deltaPercent: 0, showDelta: false };
  if (p === 0) return { deltaPercent: c > 0 ? 100 : 0, showDelta: true };
  const deltaPercent = ((c - p) / p) * 100;
  return { deltaPercent, showDelta: true };
}
