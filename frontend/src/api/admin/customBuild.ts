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
  | 'building'
  | 'staging'
  | 'validating'
  | 'awaiting_approval'
  | 'promoting'
  | 'completed'
  | 'failed'

export interface CustomUpdateStatus {
  enabled: boolean
  controller_online: boolean
  heartbeat_at?: string
  state: CustomUpdateState
  action?: 'stage' | 'promote'
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
}

export interface CustomUpdateRequestResult {
  state: 'queued'
  action: 'stage' | 'promote'
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
