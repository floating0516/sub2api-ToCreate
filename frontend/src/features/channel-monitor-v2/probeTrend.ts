/**
 * Build the /monitor availability heatmap from admin probe histories,
 * not from user usage logs.
 */
import type { MonitorTimelinePoint, UserMonitorView } from '@/api/channelMonitor'
import type { MatrixRow } from '@/components/user/monitor/MonitorStatusMatrix.vue'
import {
  MONITOR_DISPLAY_GROUPS,
  resolveDisplayGroup,
  type MonitorDisplayGroup,
} from './displayGroups'
import { formatMonitorDateTime, formatMonitorMs } from './monitorFormat'

const RANGE_MS: Record<'90m' | '24h', number> = {
  '90m': 90 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
}

const PROBE_BUCKET_MS = 30 * 60 * 1000

export function resolveProbeDisplayGroup(item: UserMonitorView): MonitorDisplayGroup | null {
  return resolveDisplayGroup(undefined, item.name || item.group_name, item.provider)
}

export function buildProbeMatrixRows(
  items: UserMonitorView[],
  range: '90m' | '24h',
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
  const buckets = new Map<string, UserMonitorView[]>()

  for (const item of items) {
    if (platforms.size && !platforms.has(item.provider)) continue
    const group = resolveProbeDisplayGroup(item)
    if (!group) continue
    if (selected.size && !selected.has(group.id)) continue
    const list = buckets.get(group.id)
    if (list) list.push(item)
    else buckets.set(group.id, [item])
  }

  return MONITOR_DISPLAY_GROUPS.filter((group) => buckets.has(group.id)).map((group) => {
    const members = buckets.get(group.id) || []
    const points = members
      .flatMap((item) => item.timeline || [])
      .filter((point) => Date.parse(point.checked_at) >= start)
      .sort((a, b) => Date.parse(a.checked_at) - Date.parse(b.checked_at))

    const availabilities = members
      .map((item) => item.availability_7d)
      .filter((value): value is number => value != null && !Number.isNaN(value))
    const latest = pickLatestMember(members)
    return {
      id: latest?.id || group.groupIds[0] || 0,
      label: labelFor(group),
      status: worstStatus(members.map((item) => item.primary_status)),
      availability: availabilities.length
        ? availabilities.reduce((sum, value) => sum + value, 0) / availabilities.length
        : probeAvailability(points),
      latency: formatMonitorMs(latest?.primary_latency_ms ?? averageNullable(points.map((point) => point.latency_ms))),
      ping: formatMonitorMs(latest?.primary_ping_latency_ms ?? averageNullable(points.map((point) => point.ping_latency_ms))),
      cells: buildAlignedCells(points, start, now, formatStatus, noSample),
    }
  })
}

function pickLatestMember(members: UserMonitorView[]): UserMonitorView | undefined {
  return [...members].sort((a, b) => {
    const aTime = Date.parse(a.timeline?.[0]?.checked_at || '') || 0
    const bTime = Date.parse(b.timeline?.[0]?.checked_at || '') || 0
    return bTime - aTime
  })[0]
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
