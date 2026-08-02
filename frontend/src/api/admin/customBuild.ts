import { apiClient } from '../client'

export interface CustomBuildNotes {
  content: string
  path: string
  updated_at: string
}

export type CustomUpdateState =
  | 'disabled'
  | 'idle'
  | 'queued'
  | 'checking'
  | 'merging'
  | 'conflict_detected'
  | 'pushing'
  | 'building'
  | 'staging'
  | 'validating'
  | 'awaiting_approval'
  | 'promoting'
  | 'completed'
  | 'failed'

export type CustomUpdateAction = 'stage' | 'promote'

export type CustomUpdateStepID =
  | 'source_check'
  | 'upstream_fetch'
  | 'upstream_merge'
  | 'conflict_resolution'
  | 'source_push'
  | 'image_build'
  | 'staging_deploy'
  | 'staging_validate'
  | 'production_approval'

export type CustomUpdateStepStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'action_required'
  | 'skipped'

export interface CustomUpdateStep {
  id: CustomUpdateStepID
  status: CustomUpdateStepStatus
}

export interface CustomUpdateStatus {
  enabled: boolean
  controller_online: boolean
  heartbeat_at?: string
  state: CustomUpdateState
  action?: CustomUpdateAction
  request_id?: string
  message?: string
  image?: string
  image_digest?: string
  app_version?: string
  upstream_commit?: string
  source_commit?: string
  started_at?: string
  updated_at?: string
  completed_at?: string
  error?: string
  log_file?: string
  staging_url?: string
  production_url?: string
  steps?: CustomUpdateStep[]
  conflict_files?: string[]
  release_status?: 'published' | 'failed'
  release_tag?: string
  release_url?: string
  release_published_at?: string
  release_error?: string
}

export interface CustomUpdateRequestResult {
  state: 'queued'
  action: CustomUpdateAction
  request_id: string
  image?: string
  message: string
}

export async function getCustomBuildNotes(): Promise<CustomBuildNotes> {
  const { data } = await apiClient.get<CustomBuildNotes>('/admin/custom-build/notes')
  return data
}

export async function getCustomUpdateStatus(): Promise<CustomUpdateStatus> {
  const { data } = await apiClient.get<CustomUpdateStatus>('/admin/custom-build/update/status')
  return data
}

export async function startCustomUpdate(): Promise<CustomUpdateRequestResult> {
  const { data } = await apiClient.post<CustomUpdateRequestResult>(
    '/admin/custom-build/update/stage'
  )
  return data
}

export async function promoteCustomUpdate(image: string): Promise<CustomUpdateRequestResult> {
  const { data } = await apiClient.post<CustomUpdateRequestResult>(
    '/admin/custom-build/update/promote',
    { image }
  )
  return data
}

export const customBuildAPI = {
  getNotes: getCustomBuildNotes,
  getUpdateStatus: getCustomUpdateStatus,
  startUpdate: startCustomUpdate,
  promoteUpdate: promoteCustomUpdate
}

export default customBuildAPI
