import { describe, expect, it } from 'vitest'
import type { MonitorHealth, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'
import {
  collapseMonitorMatrixRows,
  expandDisplayGroupIds,
  parseDisplayGroupKeys,
  resolveDisplayGroup,
} from '../displayGroups'

const health: MonitorHealth = {
  overall: 'healthy',
  error_rate: 'healthy',
  ttft: 'healthy',
  minimum_sample: 20,
}

function metrics(success: number, errors: number, errorRate = 0): MonitorMetric {
  const requestCount = success + errors
  return {
    success_requests: success,
    error_requests: errors,
    request_count: requestCount,
    token_count: 0,
    rpm: requestCount,
    tpm: 0,
    success_rate: requestCount ? success / requestCount : 0,
    error_rate: errorRate,
    cache_rate: 0,
    cache_rate_numerator: 0,
    cache_rate_denominator: 0,
    ttft: { sample_count: requestCount, p50_ms: 100, p95_ms: 200, avg_ms: 120 },
    duration: { sample_count: requestCount, p50_ms: 300, p95_ms: 400, avg_ms: 320 },
  }
}

function row(groupId: number, groupName: string, platform: string, metric: MonitorMetric): MonitorMatrixRow {
  return {
    platform,
    group_id: groupId,
    group_name: groupName,
    metrics: metric,
    health,
    buckets: [{ bucket_start: '2026-09-08T00:00:00Z', metrics: metric, health }],
  }
}

describe('monitor display groups', () => {
  it('maps known group ids onto the three product channels', () => {
    expect(resolveDisplayGroup(15)?.id).toBe('pro')
    expect(resolveDisplayGroup(2)?.id).toBe('claude_06')
    expect(resolveDisplayGroup(37)?.id).toBe('claude_opus')
    expect(resolveDisplayGroup(undefined, '公益', 'openai')).toBeNull()
  })

  it('expands selected product channels to raw group ids', () => {
    expect(expandDisplayGroupIds(['pro'])).toContain(15)
    expect(expandDisplayGroupIds(['claude_06'])).toEqual([2])
    expect(parseDisplayGroupKeys(['15', 'claude_06'])).toEqual(['pro', 'claude_06'])
  })

  it('collapses raw groups into three labeled matrix rows', () => {
    const collapsed = collapseMonitorMatrixRows([
      row(15, 'GPT_0.14', 'openai', metrics(90, 10)),
      row(18, 'pro拼车', 'openai', metrics(10, 10, 0.1)),
      row(2, 'claude_0.6', 'anthropic', metrics(50, 0)),
      row(37, 'claude-cursor逆向', 'anthropic', metrics(0, 5, 1)),
    ], 'platform_group')

    expect(collapsed.map((item) => item.group_name)).toEqual(['Pro 渠道', 'Claude Opus', 'Claude 0.6'])
    expect(collapsed[0].metrics.request_count).toBe(120)
    expect(collapsed[0].metrics.success_rate).toBeCloseTo(100 / 120)
    expect(collapsed[1].metrics.success_rate).toBe(0)
    expect(collapsed[2].metrics.success_rate).toBe(1)
  })
})
