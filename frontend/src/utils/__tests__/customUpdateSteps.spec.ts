import { describe, expect, it } from 'vitest'
import { resolveCustomUpdateSteps } from '../customUpdateSteps'

describe('resolveCustomUpdateSteps', () => {
  it('preserves reported controller results and fills missing steps as pending', () => {
    const steps = resolveCustomUpdateSteps({
      state: 'building',
      steps: [
        { id: 'source_check', status: 'completed' },
        { id: 'image_build', status: 'running' }
      ]
    })

    expect(steps).toHaveLength(9)
    expect(steps[0]).toEqual({ id: 'source_check', status: 'completed' })
    expect(steps[3]).toEqual({ id: 'conflict_resolution', status: 'skipped' })
    expect(steps[5]).toEqual({ id: 'image_build', status: 'running' })
    expect(steps[8]).toEqual({ id: 'production_approval', status: 'pending' })
  })

  it('preserves the reported conflict-resolution state from the new controller', () => {
    const steps = resolveCustomUpdateSteps({
      state: 'ai_resolving',
      steps: [{ id: 'conflict_resolution', status: 'running' }]
    })

    expect(steps[3]).toEqual({ id: 'conflict_resolution', status: 'running' })
  })

  it('maps an older awaiting-approval status to eight completed steps', () => {
    const steps = resolveCustomUpdateSteps({ state: 'awaiting_approval' })

    expect(steps.slice(0, 8).every((step) => step.status === 'completed')).toBe(true)
    expect(steps[8]).toEqual({
      id: 'production_approval',
      status: 'action_required'
    })
  })

  it('shows source push as the active legacy step', () => {
    const steps = resolveCustomUpdateSteps({ state: 'pushing' })

    expect(steps.slice(0, 4).every((step) => step.status === 'completed')).toBe(true)
    expect(steps[4]).toEqual({ id: 'source_push', status: 'running' })
  })

  it('marks AI conflict resolution as requiring review', () => {
    const steps = resolveCustomUpdateSteps({ state: 'resolution_ready' })

    expect(steps.slice(0, 3).every((step) => step.status === 'completed')).toBe(true)
    expect(steps[3]).toEqual({
      id: 'conflict_resolution',
      status: 'action_required'
    })
  })
})
