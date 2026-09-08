/**
 * Product-level channel rows on /monitor.
 *
 * Edit MONITOR_DISPLAY_GROUPS to add/remove rows, or set
 * MONITOR_DISPLAY_GROUP_COLLAPSE = false to show every raw group again.
 */
import type { MonitorHealth, MonitorMatrixGroupBy, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'

export const MONITOR_DISPLAY_GROUP_COLLAPSE = true

export interface MonitorDisplayGroup {
  id: 'pro' | 'claude_opus' | 'claude_06'
  labelKey: string
  /** Fallback label when i18n is unavailable (tests / backend-aligned names). */
  label: string
  groupIds: number[]
  platforms: string[]
}

export const MONITOR_DISPLAY_GROUPS: MonitorDisplayGroup[] = [
  {
    id: 'pro',
    labelKey: 'channelMonitorV2.displayGroups.pro',
    label: 'Pro 渠道',
    groupIds: [15, 18, 21, 27, 29, 30, 31],
    platforms: ['openai'],
  },
  {
    id: 'claude_opus',
    labelKey: 'channelMonitorV2.displayGroups.claudeOpus',
    label: 'Claude Opus',
    groupIds: [12, 16, 35, 37],
    platforms: ['anthropic'],
  },
  {
    id: 'claude_06',
    labelKey: 'channelMonitorV2.displayGroups.claude06',
    label: 'Claude 0.6',
    groupIds: [2],
    platforms: ['anthropic'],
  },
]

const byId = new Map(MONITOR_DISPLAY_GROUPS.map((group) => [group.id, group]))

export function isDisplayGroupId(value: string): value is MonitorDisplayGroup['id'] {
  return byId.has(value as MonitorDisplayGroup['id'])
}

export function expandDisplayGroupIds(keys: string[]): number[] {
  const ids = new Set<number>()
  for (const key of keys) {
    const group = byId.get(key as MonitorDisplayGroup['id'])
    if (!group) continue
    for (const id of group.groupIds) ids.add(id)
  }
  return [...ids]
}

export function parseDisplayGroupKeys(raw: string[]): string[] {
  const keys = new Set<string>()
  for (const value of raw) {
    if (isDisplayGroupId(value)) {
      keys.add(value)
      continue
    }
    const id = Number(value)
    if (!Number.isInteger(id) || id <= 0) continue
    const group = resolveDisplayGroup(id)
    if (group) keys.add(group.id)
  }
  return MONITOR_DISPLAY_GROUPS.map((group) => group.id).filter((id) => keys.has(id))
}

export function resolveDisplayGroup(
  groupId?: number | null,
  groupName?: string,
  platform?: string,
): MonitorDisplayGroup | null {
  if (!MONITOR_DISPLAY_GROUP_COLLAPSE) return null
  if (groupId && groupId > 0) {
    const hit = MONITOR_DISPLAY_GROUPS.find((group) => group.groupIds.includes(groupId))
    if (hit) return hit
  }
  const name = (groupName || '').toLowerCase()
  const plat = (platform || '').toLowerCase()
  if (/0\.6/.test(name) || name.includes('claude_0.6')) {
    return byId.get('claude_06') || null
  }
  if (/公益|public|free/.test(name)) return null
  if (plat === 'openai' || /gpt|\bpro\b|拼车/.test(name)) {
    return byId.get('pro') || null
  }
  if (plat === 'anthropic' || /claude|opus/.test(name)) {
    return byId.get('claude_opus') || null
  }
  return null
}

export function collapseMonitorMatrixRows(
  rows: MonitorMatrixRow[],
  groupBy: MonitorMatrixGroupBy,
  labelFor: (group: MonitorDisplayGroup) => string = (group) => group.label,
): MonitorMatrixRow[] {
  if (!MONITOR_DISPLAY_GROUP_COLLAPSE) return rows
  if (groupBy !== 'platform_group' && groupBy !== 'platform_group_model') return rows

  const buckets = new Map<string, MonitorMatrixRow[]>()
  for (const row of rows) {
    const group = resolveDisplayGroup(row.group_id, row.group_name, row.platform)
    if (!group) continue
    const key = groupBy === 'platform_group_model'
      ? `${group.id}::${row.model || ''}`
      : group.id
    const labeled: MonitorMatrixRow = {
      ...row,
      platform: '',
      group_id: group.groupIds[0],
      group_name: labelFor(group),
    }
    const list = buckets.get(key)
    if (list) list.push(labeled)
    else buckets.set(key, [labeled])
  }

  const ordered: MonitorMatrixRow[] = []
  for (const group of MONITOR_DISPLAY_GROUPS) {
    if (groupBy === 'platform_group') {
      const list = buckets.get(group.id)
      if (list) ordered.push(mergeMatrixRows(list))
      continue
    }
    const prefix = `${group.id}::`
    const modelKeys = [...buckets.keys()].filter((key) => key.startsWith(prefix)).sort()
    for (const key of modelKeys) {
      ordered.push(mergeMatrixRows(buckets.get(key) || []))
    }
  }
  return ordered
}

function mergeMatrixRows(rows: MonitorMatrixRow[]): MonitorMatrixRow {
  const first = rows[0]
  if (rows.length === 1) return first
  const bucketMap = new Map<string, MonitorMatrixRow['buckets'][number][]>()
  for (const row of rows) {
    for (const bucket of row.buckets || []) {
      const list = bucketMap.get(bucket.bucket_start)
      if (list) list.push(bucket)
      else bucketMap.set(bucket.bucket_start, [bucket])
    }
  }
  const starts = [...bucketMap.keys()].sort()
  return {
    platform: first.platform,
    group_id: first.group_id,
    group_name: first.group_name,
    model: first.model,
    metrics: mergeMetrics(rows.map((row) => row.metrics)),
    health: mergeHealth(rows.map((row) => row.health)),
    buckets: starts.map((start) => {
      const list = bucketMap.get(start) || []
      return {
        bucket_start: start,
        metrics: mergeMetrics(list.map((item) => item.metrics)),
        health: mergeHealth(list.map((item) => item.health)),
      }
    }),
  }
}

function mergeMetrics(list: MonitorMetric[]): MonitorMetric {
  const weights = metricWeights(list)
  const requestCount = list.reduce((sum, item) => sum + (item.request_count || 0), 0)
  const successRequests = list.reduce((sum, item) => sum + (item.success_requests || 0), 0)
  const errorRequests = list.reduce((sum, item) => sum + (item.error_requests || 0), 0)
  const successRate = requestCount > 0
    ? successRequests / requestCount
    : weightedAverage(list, weights, (item) => item.success_rate ?? 1 - (item.error_rate || 0))
  return {
    success_requests: successRequests,
    error_requests: errorRequests,
    request_count: requestCount,
    token_count: list.reduce((sum, item) => sum + (item.token_count || 0), 0),
    rpm: list.reduce((sum, item) => sum + (item.rpm || 0), 0),
    tpm: list.reduce((sum, item) => sum + (item.tpm || 0), 0),
    success_rate: successRate,
    error_rate: weightedAverage(list, weights, (item) => item.error_rate || 0),
    cache_rate: weightedAverage(list, weights, (item) => item.cache_rate || 0),
    cache_rate_numerator: list.reduce((sum, item) => sum + (item.cache_rate_numerator || 0), 0),
    cache_rate_denominator: list.reduce((sum, item) => sum + (item.cache_rate_denominator || 0), 0),
    ttft: mergeLatency(list.map((item) => item.ttft), weights),
    duration: mergeLatency(list.map((item) => item.duration), weights),
    upstream_affected_requests: sumOptional(list.map((item) => item.upstream_affected_requests)),
    upstream_attempt_count: sumOptional(list.map((item) => item.upstream_attempt_count)),
  }
}

function metricWeights(list: MonitorMetric[]): number[] {
  const counts = list.map((item) => {
    if ((item.request_count || 0) > 0) return item.request_count
    if ((item.rpm || 0) > 0) return item.rpm
    return 0
  })
  if (counts.some((value) => value > 0)) return counts
  return list.map(() => 1)
}

function weightedAverage(
  list: MonitorMetric[],
  weights: number[],
  read: (item: MonitorMetric) => number,
): number {
  let num = 0
  let den = 0
  list.forEach((item, index) => {
    const weight = weights[index] || 0
    num += read(item) * weight
    den += weight
  })
  return den > 0 ? num / den : 0
}

function mergeLatency(
  list: MonitorMetric['ttft'][],
  weights: number[],
): MonitorMetric['ttft'] {
  const sampleCount = list.reduce((sum, item) => sum + (item?.sample_count || 0), 0)
  const avg = weightedNullable(list.map((item) => item?.avg_ms ?? null), weights)
  return {
    sample_count: sampleCount,
    p50_ms: weightedNullable(list.map((item) => item?.p50_ms ?? null), weights),
    p90_ms: weightedNullable(list.map((item) => item?.p90_ms ?? null), weights),
    p95_ms: weightedNullable(list.map((item) => item?.p95_ms ?? null), weights),
    avg_ms: avg,
  }
}

function weightedNullable(values: Array<number | null>, weights: number[]): number | null {
  let num = 0
  let den = 0
  values.forEach((value, index) => {
    if (value == null || Number.isNaN(value)) return
    const weight = weights[index] || 0
    num += value * weight
    den += weight
  })
  return den > 0 ? num / den : null
}

function mergeHealth(list: MonitorHealth[]): MonitorHealth {
  const first = list[0]
  const scores = list.map((item) => item.score).filter((value): value is number => value != null)
  const errorScores = list.map((item) => item.error_rate_score).filter((value): value is number => value != null)
  const ttftScores = list.map((item) => item.ttft_score).filter((value): value is number => value != null)
  const cacheScores = list.map((item) => item.cache_score).filter((value): value is number => value != null)
  return {
    ...first,
    overall: worstState(list.map((item) => item.overall)),
    error_rate: worstState(list.map((item) => item.error_rate)),
    ttft: worstState(list.map((item) => item.ttft)),
    cache: worstState(list.map((item) => item.cache || 'unknown')),
    score: scores.length ? Math.min(...scores) : first?.score,
    error_rate_score: errorScores.length ? Math.min(...errorScores) : first?.error_rate_score,
    ttft_score: ttftScores.length ? Math.min(...ttftScores) : first?.ttft_score,
    cache_score: cacheScores.length ? Math.min(...cacheScores) : first?.cache_score,
  }
}

function worstState(states: string[]): MonitorHealth['overall'] {
  const rank: Record<string, number> = { critical: 4, warning: 3, healthy: 2, unknown: 1 }
  let worst = 'unknown'
  for (const state of states) {
    if ((rank[state] || 0) > (rank[worst] || 0)) worst = state
  }
  return worst as MonitorHealth['overall']
}

function sumOptional(values: Array<number | undefined>): number | undefined {
  const present = values.filter((value): value is number => value != null)
  if (!present.length) return undefined
  return present.reduce((sum, value) => sum + value, 0)
}
