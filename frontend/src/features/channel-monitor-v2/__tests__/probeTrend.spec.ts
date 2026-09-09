import { describe, expect, it } from 'vitest'
import type { UserMonitorView } from '@/api/channelMonitor'
import { buildProbeMatrixRows, probeRowLabel, resolveProbeDisplayGroup } from '../probeTrend'

function monitor(partial: Partial<UserMonitorView> & Pick<UserMonitorView, 'id' | 'name' | 'provider'>): UserMonitorView {
  return {
    group_name: '',
    primary_model: 'gpt-5',
    primary_status: 'operational',
    primary_latency_ms: 200,
    primary_ping_latency_ms: 40,
    availability_7d: 99,
    extra_models: [],
    timeline: [],
    ...partial,
  }
}

describe('probeTrend', () => {
  it('maps the three live probe names onto product channels', () => {
    expect(resolveProbeDisplayGroup(monitor({ id: 4, name: 'GPT-PRO', provider: 'openai' }))?.id).toBe('pro')
    expect(resolveProbeDisplayGroup(monitor({ id: 9, name: 'Claude x Cursor', provider: 'anthropic' }))?.id).toBe('claude_opus')
    expect(resolveProbeDisplayGroup(monitor({ id: 7, name: 'claude_0.6', provider: 'anthropic' }))?.id).toBe('claude_06')
  })

  it('uses the configured probe name for a newly added monitor', () => {
    const extra = monitor({ id: 12, name: 'Gemini Flash', provider: 'gemini' })
    expect(probeRowLabel(extra, (group) => group.label)).toBe('Gemini Flash')
  })

  it('builds one row per enabled probe and grows when a new probe is added', () => {
    const now = Date.parse('2026-09-08T14:00:00Z')
    const current = [
      monitor({
        id: 4,
        name: 'GPT-PRO',
        provider: 'openai',
        timeline: [
          { status: 'operational', latency_ms: 180, ping_latency_ms: 30, checked_at: '2026-09-08T13:30:00Z' },
          { status: 'failed', latency_ms: 800, ping_latency_ms: 40, checked_at: '2026-09-08T13:00:00Z' },
        ],
      }),
      monitor({
        id: 7,
        name: 'claude_0.6',
        provider: 'anthropic',
        timeline: [
          { status: 'operational', latency_ms: 220, ping_latency_ms: 35, checked_at: '2026-09-08T13:30:00Z' },
        ],
      }),
      monitor({
        id: 9,
        name: 'Claude x Cursor',
        provider: 'anthropic',
        primary_status: 'degraded',
        timeline: [
          { status: 'degraded', latency_ms: 400, ping_latency_ms: 50, checked_at: '2026-09-08T13:30:00Z' },
        ],
      }),
    ]

    const currentRows = buildProbeMatrixRows(
      current,
      '90m',
      [],
      [],
      (group) => group.label,
      (status) => status,
      '无探测',
      now,
    )
    expect(currentRows.map((row) => row.label)).toEqual(['Pro 渠道', 'Claude Opus', 'Claude 0.6'])
    expect(currentRows[0].cells.some((cell) => cell.status === 'failed')).toBe(true)
    expect(currentRows[1].status).toBe('degraded')
    expect(currentRows[2].cells.some((cell) => cell.status === 'operational')).toBe(true)

    const grown = buildProbeMatrixRows(
      [
        ...current,
        monitor({
          id: 12,
          name: 'Gemini Flash',
          provider: 'gemini',
          timeline: [
            { status: 'operational', latency_ms: 260, ping_latency_ms: 20, checked_at: '2026-09-08T13:30:00Z' },
          ],
        }),
      ],
      '90m',
      [],
      [],
      (group) => group.label,
      (status) => status,
      '无探测',
      now,
    )
    expect(grown.map((row) => row.label)).toEqual(['Pro 渠道', 'Claude Opus', 'Claude 0.6', 'Gemini Flash'])
    expect(grown[3].id).toBe(12)
  })
})
