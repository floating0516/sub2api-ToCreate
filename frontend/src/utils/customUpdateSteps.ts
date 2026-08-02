import type {
  CustomUpdateState,
  CustomUpdateStatus,
  CustomUpdateStep,
  CustomUpdateStepID,
  CustomUpdateStepStatus
} from '@/api/admin/customBuild'

export const CUSTOM_UPDATE_STEP_IDS = [
  'source_check',
  'upstream_fetch',
  'upstream_merge',
  'conflict_resolution',
  'source_push',
  'image_build',
  'staging_deploy',
  'staging_validate',
  'production_approval'
] as const satisfies readonly CustomUpdateStepID[]

const validStepStatuses = new Set<CustomUpdateStepStatus>([
  'pending',
  'running',
  'completed',
  'failed',
  'action_required',
  'skipped'
])

const legacyStateProgress: Record<
  CustomUpdateState,
  { completed: number; active?: number; activeStatus?: CustomUpdateStepStatus }
> = {
  disabled: { completed: 0 },
  idle: { completed: 0 },
  queued: { completed: 0 },
  checking: { completed: 0, active: 0, activeStatus: 'running' },
  merging: { completed: 2, active: 2, activeStatus: 'running' },
  conflict_detected: { completed: 3, active: 3, activeStatus: 'action_required' },
  pushing: { completed: 4, active: 4, activeStatus: 'running' },
  building: { completed: 5, active: 5, activeStatus: 'running' },
  staging: { completed: 6, active: 6, activeStatus: 'running' },
  validating: { completed: 7, active: 7, activeStatus: 'running' },
  awaiting_approval: { completed: 8, active: 8, activeStatus: 'action_required' },
  promoting: { completed: 9 },
  completed: { completed: 9 },
  failed: { completed: 0 }
}

function resolveReportedSteps(steps: CustomUpdateStep[]): CustomUpdateStep[] {
  const reported = new Map<CustomUpdateStepID, CustomUpdateStepStatus>()
  for (const step of steps) {
    if (
      CUSTOM_UPDATE_STEP_IDS.includes(step.id) &&
      validStepStatuses.has(step.status)
    ) {
      reported.set(step.id, step.status)
    }
  }

  const legacyConflictStepMissing =
    !reported.has('conflict_resolution') &&
    CUSTOM_UPDATE_STEP_IDS.slice(4).some((id) => reported.has(id))

  return CUSTOM_UPDATE_STEP_IDS.map((id) => ({
    id,
    status:
      reported.get(id) ||
      (id === 'conflict_resolution' && legacyConflictStepMissing ? 'skipped' : 'pending')
  }))
}

function resolveLegacySteps(state: CustomUpdateState): CustomUpdateStep[] {
  const progress = legacyStateProgress[state]
  return CUSTOM_UPDATE_STEP_IDS.map((id, index) => {
    let status: CustomUpdateStepStatus = 'pending'
    if (index < progress.completed) {
      status = 'completed'
    } else if (index === progress.active) {
      status = progress.activeStatus || 'running'
    }
    return { id, status }
  })
}

export function resolveCustomUpdateSteps(
  status: Pick<CustomUpdateStatus, 'state' | 'steps'> | null | undefined
): CustomUpdateStep[] {
  if (!status) {
    return resolveLegacySteps('idle')
  }
  if (status.steps?.length) {
    return resolveReportedSteps(status.steps)
  }
  return resolveLegacySteps(status.state)
}
