/**
 * Build the /monitor availability heatmap from admin probe histories,
 * not from user usage logs.
 *
 * One enabled probe = one row. Adding a monitor later grows the matrix
 * without another frontend code change.
 */
import type { MonitorTimelinePoint, UserMonitorView } from '@/api/channelMonitor'
import type { MonitorRange } from '@/api/channelMonitorV2'
import type { MatrixRow } from '@/components/user/monitor/MonitorStatusMatrix.vue'
import {
  MONITOR_DISPLAY_GROUPS,
  resolveDisplayGroup,
  type MonitorDisplayGroup,
} from './displayGroups'
import { formatMonitorDateTime, formatMonitorMs } from './monitorFormat'

const RANGE_MS: Record<MonitorRange, number> = {
  '90m': 90 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
  '7d': 24 * 60 * 60 * 1000,
  '30d': 24 * 60 * 60 * 1000,
}

const PROBE_BUCKET_MS = 30 * 60 * 1000

/** Friendly labels for the current product probes; anything else uses the monitor name. */
const PROBE_LABEL_ALIASES: Record<string, MonitorDisplayGroup['id']> = {
  'gpt-pro': 'pro',
  'claude x cursor': 'claude_opus',
  'claude_0.6': 'claude_06',
}

export function resolveProbeDisplayGroup(item: UserMonitorView): MonitorDisplayGroup | null {
  const aliasId = PROBE_LABEL_ALIASES[normalizeProbeName(item.name)]
  if (aliasId) {
    return MONITOR_DISPLAY_GROUPS.find((group) => group.id === aliasId) || null
  }
  return resolveDisplayGroup(undefined, item.name || item.group_name, item.provider)
}

export function probeRowLabel(
  item: UserMonitorView,
  labelFor: (group: MonitorDisplayGroup) => string,
): string {
  const aliasId = PROBE_LABEL_ALIASES[normalizeProbeName(item.name)]
  if (aliasId) {
    const group = MONITOR_DISPLAY_GROUPS.find((entry) => entry.id === aliasId)
    if (group) return labelFor(group)
  }
  return (item.name || item.group_name || '').trim() || `探测 ${item.id}`
}

export function buildProbeMatrixRows(
  items: UserMonitorView[],
  range: MonitorRange,
  selectedGroupKeys: string[],
  selectedPlatforms: string[],
  labelFor: (group: MonitorDisplayGroup) => string,
  formatStatus: (status: string) => string,
  noSample: string,
  now = Date.now(),
): MatrixRow[] {
  const start = now - RANGE_MS[range]
  const selected = new Set(selectedGroupKeys)
  const platforms = new Set(selectedPlatforms)

  const visible = items.filter((item) => {
    if (platforms.size && !platforms.has(item.provider)) return false
    if (!selected.size) return true
    const group = resolveProbeDisplayGroup(item)
    return Boolean(group && selected.has(group.id))
  })

  visible.sort((a, b) => {
    const rankDiff = probeSortRank(a) - probeSortRank(b)
    if (rankDiff !== 0) return rankDiff
    return probeRowLabel(a, (group) => group.label).localeCompare(
      probeRowLabel(b, (group) => group.label),
      'zh',
    )
  })

  return visible.map((item) => {
    const points = (item.timeline || [])
      .filter((point) => Date.parse(point.checked_at) >= start)
      .sort((a, b) => Date.parse(a.checked_at) - Date.parse(b.checked_at))
    return {
      id: item.id,
      label: probeRowLabel(item, labelFor),
      status: worstStatus([item.primary_status, ...points.map((point) => point.status)]),
      availability: item.availability_7d != null && !Number.isNaN(item.availability_7d)
        ? item.availability_7d
        : probeAvailability(points),
      latency: formatMonitorMs(item.primary_latency_ms ?? averageNullable(points.map((point) => point.latency_ms))),
      ping: formatMonitorMs(item.primary_ping_latency_ms ?? averageNullable(points.map((point) => point.ping_latency_ms))),
      cells: buildAlignedCells(points, start, now, formatStatus, noSample),
    }
  })
}

function normalizeProbeName(name?: string): string {
  return (name || '').trim().toLowerCase()
}

function probeSortRank(item: UserMonitorView): number {
  const aliasId = PROBE_LABEL_ALIASES[normalizeProbeName(item.name)]
  if (aliasId) {
    const index = MONITOR_DISPLAY_GROUPS.findIndex((group) => group.id === aliasId)
    if (index >= 0) return index
  }
  return MONITOR_DISPLAY_GROUPS.length + item.id
}

function buildAlignedCells(
  points: MonitorTimelinePoint[],
  start: number,
  now: number,
  formatStatus: (status: string) => string,
  noSample: string,
) {
  const alignedStart = Math.floor(start / PROBE_BUCKET_MS) * PROBE_BUCKET_MS
  const cells = []
  for (let cursor = alignedStart; cursor < now; cursor += PROBE_BUCKET_MS) {
    const inBucket = points.filter((point) => {
      const ts = Date.parse(point.checked_at)
      return ts >= cursor && ts < cursor + PROBE_BUCKET_MS
    })
    if (!inBucket.length) {
      cells.push({
        status: 'empty',
        title: noSample,
        lines: [formatMonitorDateTime(new Date(cursor)), noSample],
      })
      continue
    }
    const status = worstStatus(inBucket.map((point) => point.status))
    cells.push({
      status,
      title: `${formatMonitorDateTime(new Date(cursor))} · ${formatStatus(status)}`,
      lines: [
        formatMonitorDateTime(new Date(cursor)),
        formatStatus(status),
        `延迟 ${formatMonitorMs(averageNullable(inBucket.map((point) => point.latency_ms)))}`,
        `Ping ${formatMonitorMs(averageNullable(inBucket.map((point) => point.ping_latency_ms)))}`,
      ],
    })
  }
  return cells
}

function worstStatus(statuses: string[]): string {
  const rank: Record<string, number> = {
    failed: 4,
    error: 4,
    degraded: 3,
    operational: 2,
    empty: 1,
  }
  let worst = 'empty'
  for (const status of statuses) {
    if ((rank[status] || 0) > (rank[worst] || 0)) worst = status
  }
  return worst
}

function averageNullable(values: Array<number | null | undefined>): number | null {
  const present = values.filter((value): value is number => value != null && !Number.isNaN(value))
  if (!present.length) return null
  return present.reduce((sum, value) => sum + value, 0) / present.length
}

function probeAvailability(points: MonitorTimelinePoint[]): number | null {
  if (!points.length) return null
  const ok = points.filter((point) => point.status === 'operational').length
  return (ok / points.length) * 100
}
