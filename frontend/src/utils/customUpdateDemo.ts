import type { CustomUpdateStatus } from '@/api/admin/customBuild'

export const CUSTOM_UPDATE_DEMO_QUERY_PARAM = 'custom_update_demo'

export const CUSTOM_UPDATE_DEMO_SCENARIOS = [
  'conflict',
  'resolving',
  'review',
  'review_high',
  'failed',
  'staged'
] as const

export type CustomUpdateDemoScenario = (typeof CUSTOM_UPDATE_DEMO_SCENARIOS)[number]

export interface CustomUpdateDemoCopy {
  reviewSummary: string
  highRiskSummary: string
  reviewWarnings: string[]
  highRiskWarnings: string[]
  failedError: string
}

const conflictFiles = [
  'frontend/src/components/common/VersionBadge.vue',
  'backend/internal/server/routes/admin.go',
  'deploy/tocreate-update-controller.sh'
]

const diffStat = [
  ' frontend/src/components/common/VersionBadge.vue | 48 ++++++++++------',
  ' backend/internal/server/routes/admin.go           | 10 +++-',
  ' deploy/tocreate-update-controller.sh              | 92 ++++++++++++++++++---',
  ' 3 files changed, 119 insertions(+), 31 deletions(-)'
].join('\n')

export function isCustomUpdateDemoScenario(
  value: string | null
): value is CustomUpdateDemoScenario {
  return CUSTOM_UPDATE_DEMO_SCENARIOS.includes(value as CustomUpdateDemoScenario)
}

export function resolveCustomUpdateDemoScenario(
  search: string,
  port: string
): CustomUpdateDemoScenario | null {
  if (port !== '18080') return null
  const value = new URLSearchParams(search).get(CUSTOM_UPDATE_DEMO_QUERY_PARAM)
  return isCustomUpdateDemoScenario(value) ? value : null
}

export function createCustomUpdateDemoUrl(
  href: string,
  scenario: CustomUpdateDemoScenario
): string {
  const url = new URL(href)
  url.pathname = '/admin/custom-build'
  url.search = ''
  url.searchParams.set(CUSTOM_UPDATE_DEMO_QUERY_PARAM, scenario)
  url.hash = ''
  return url.toString()
}

export function createCustomUpdateDemoStatus(
  scenario: CustomUpdateDemoScenario,
  copy: CustomUpdateDemoCopy,
  now = new Date()
): CustomUpdateStatus {
  const updatedAt = now.toISOString()
  const common: CustomUpdateStatus = {
    enabled: true,
    controller_online: true,
    heartbeat_at: updatedAt,
    state: 'conflict_detected',
    action: 'stage',
    request_id: 'demo0000000000000000000000000000',
    upstream_commit: '2980ff385076593d0db9aaa30c590db153e27366',
    source_commit: '7528de1c76e45ac0cbd3d025ac49a796176e3062',
    started_at: new Date(now.getTime() - 90_000).toISOString(),
    updated_at: updatedAt,
    staging_url: 'http://127.0.0.1:18080',
    production_url: 'http://127.0.0.1:8080',
    conflict_files: [...conflictFiles]
  }

  switch (scenario) {
    case 'conflict':
      return common
    case 'resolving':
      return {
        ...common,
        state: 'ai_resolving',
        resolver_model: 'gpt-5.6-luna'
      }
    case 'review':
      return {
        ...common,
        state: 'resolution_ready',
        resolution_id: '11111111111111111111111111111111',
        resolver_model: 'gpt-5.6-luna',
        resolution_summary: copy.reviewSummary,
        resolution_risk_level: 'medium',
        resolution_warnings: [...copy.reviewWarnings],
        resolution_diff_stat: diffStat
      }
    case 'review_high':
      return {
        ...common,
        state: 'resolution_ready',
        resolution_id: '22222222222222222222222222222222',
        resolver_model: 'gpt-5.6-luna',
        resolution_summary: copy.highRiskSummary,
        resolution_risk_level: 'high',
        resolution_warnings: [...copy.highRiskWarnings],
        resolution_diff_stat: diffStat
      }
    case 'failed':
      return {
        ...common,
        state: 'resolution_failed',
        resolution_id: '33333333333333333333333333333333',
        resolver_model: 'gpt-5.6-luna',
        error: copy.failedError
      }
    case 'staged':
      return {
        ...common,
        state: 'awaiting_approval',
        conflict_files: [],
        image: 'ghcr.io/floating0516/sub2api-tocreate:0.1.170-tc1.24-rc.1',
        image_digest:
          'sha256:4e3d2523f940ac6e4ba3ab2e57b769680b0f04d72c0289aa0b6c23943a59c117',
        app_version: '0.1.170'
      }
  }
}
