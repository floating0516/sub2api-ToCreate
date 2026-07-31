import { describe, expect, it } from 'vitest'
import {
  CUSTOM_UPDATE_DEMO_SCENARIOS,
  createCustomUpdateDemoStatus,
  resolveCustomUpdateDemoScenario,
  type CustomUpdateDemoCopy
} from '../customUpdateDemo'

const copy: CustomUpdateDemoCopy = {
  reviewSummary: 'review summary',
  highRiskSummary: 'high risk summary',
  reviewWarnings: ['review warning'],
  highRiskWarnings: ['high risk warning'],
  failedError: 'resolver failed'
}

describe('custom update demo fixtures', () => {
  it('only enables a supported scenario on staging port 18080', () => {
    expect(resolveCustomUpdateDemoScenario('?custom_update_demo=review', '18080')).toBe('review')
    expect(resolveCustomUpdateDemoScenario('?custom_update_demo=unknown', '18080')).toBeNull()
    expect(resolveCustomUpdateDemoScenario('?custom_update_demo=review', '8080')).toBeNull()
  })

  it('creates enabled, online status fixtures for every scenario', () => {
    for (const scenario of CUSTOM_UPDATE_DEMO_SCENARIOS) {
      const status = createCustomUpdateDemoStatus(
        scenario,
        copy,
        new Date('2026-08-01T00:00:00Z')
      )
      expect(status.enabled).toBe(true)
      expect(status.controller_online).toBe(true)
      expect(status.updated_at).toBe('2026-08-01T00:00:00.000Z')
    }
  })

  it('includes review metadata without using a real resolution ID', () => {
    const status = createCustomUpdateDemoStatus('review_high', copy)

    expect(status.state).toBe('resolution_ready')
    expect(status.resolution_id).toHaveLength(32)
    expect(status.resolution_risk_level).toBe('high')
    expect(status.resolver_model).toBe('gpt-5.6-luna')
  })

  it('shows a staged candidate without conflict details', () => {
    const status = createCustomUpdateDemoStatus('staged', copy)

    expect(status.state).toBe('awaiting_approval')
    expect(status.conflict_files).toEqual([])
    expect(status.image).toContain('-rc.1')
  })
})
