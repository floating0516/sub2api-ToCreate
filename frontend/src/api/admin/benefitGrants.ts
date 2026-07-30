import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

const benefitGrantExecutionTimeoutMs = 90000

export type BenefitGrantAudienceType =
  | 'today_active'
  | 'recent_active'
  | 'recent_registered'
  | 'authenticated_activity'

export type BenefitGrantDeliveryMode = 'snapshot' | 'activity_window'
export type BenefitGrantType = 'subscription' | 'balance'
export type BenefitGrantConflictPolicy = 'skip_active' | 'extend_active' | 'none'
export type BenefitGrantCampaignStatus =
  | 'scheduled'
  | 'running'
  | 'completed'
  | 'partial'
  | 'failed'
export type BenefitGrantAnnouncementNotifyMode = 'silent' | 'popup'
export type BenefitGrantRecipientStatus =
  | 'pending'
  | 'processing'
  | 'granted'
  | 'skipped'
  | 'failed'

export interface BenefitGrantRequest {
  operation_key: string
  audience_type: BenefitGrantAudienceType
  audience_date: string
  audience_days: number
  timezone: string
  benefit_type: BenefitGrantType
  conflict_policy?: BenefitGrantConflictPolicy
  group_id?: number
  validity_days?: number
  balance_amount?: number
  notes?: string
  announcement_enabled?: boolean
  announcement_title?: string
  announcement_content?: string
  announcement_notify_mode?: BenefitGrantAnnouncementNotifyMode
}

export interface AutomaticBenefitGrantRequest {
  operation_key: string
  timezone: string
  window_start: number
  window_end: number
  benefit_type: BenefitGrantType
  conflict_policy?: BenefitGrantConflictPolicy
  group_id?: number
  validity_days?: number
  balance_amount?: number
  notes?: string
  announcement_enabled?: boolean
  announcement_title?: string
  announcement_content?: string
  announcement_notify_mode?: BenefitGrantAnnouncementNotifyMode
}

export interface BenefitGrantPreview {
  operation_key: string
  audience_type: BenefitGrantAudienceType
  audience_date: string
  audience_days: number
  timezone: string
  window_start: string
  window_end: string
  benefit_type: BenefitGrantType
  conflict_policy: BenefitGrantConflictPolicy
  group_id: number
  validity_days: number
  balance_amount: number
  matched_count: number
  eligible_count: number
  already_granted_count: number
  conflict_count: number
  snapshot_token: string
}

export interface BenefitGrantCampaign {
  id: number
  delivery_mode: BenefitGrantDeliveryMode
  audience_type: BenefitGrantAudienceType
  audience_date: string
  audience_days: number
  timezone: string
  window_start: string
  window_end: string
  benefit_type: BenefitGrantType
  conflict_policy: BenefitGrantConflictPolicy
  group_id?: number
  group_name?: string
  validity_days?: number
  balance_amount?: number
  notes: string
  announcement_id?: number
  announcement_title?: string
  announcement_content?: string
  announcement_notify_mode?: BenefitGrantAnnouncementNotifyMode
  status: BenefitGrantCampaignStatus
  matched_count: number
  eligible_count: number
  already_granted_count: number
  conflict_count: number
  granted_count: number
  skipped_count: number
  failed_count: number
  created_count: number
  renewed_count: number
  extended_count: number
  balance_granted_count: number
  created_by: number
  started_at: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface BenefitGrantRecipient {
  id: number
  campaign_id: number
  user_id: number
  email: string
  username: string
  eligibility: 'eligible' | 'already_granted' | 'conflict'
  planned_action: string
  status: BenefitGrantRecipientStatus
  result_type?: string
  subscription_id?: number
  balance_before?: number
  balance_after?: number
  error?: string
  attempt_count: number
  last_attempt_at?: string
  created_at: string
  updated_at: string
}

export interface BenefitGrantResult {
  campaign: BenefitGrantCampaign
  preview?: BenefitGrantPreview
  granted_count: number
  created_count: number
  renewed_count: number
  extended_count: number
  failed_count: number
  skipped_count: number
  errors: string[]
}

export async function preview(request: BenefitGrantRequest): Promise<BenefitGrantPreview> {
  const { data } = await apiClient.post<BenefitGrantPreview>(
    '/admin/benefit-grants/preview',
    request
  )
  return data
}

export async function execute(
  request: BenefitGrantRequest & {
    expected_matched_count: number
    expected_eligible_count: number
    expected_snapshot: string
  }
): Promise<BenefitGrantResult> {
  const { data } = await apiClient.post<BenefitGrantResult>(
    '/admin/benefit-grants/execute',
    request,
    {
      headers: {
        'Idempotency-Key': request.operation_key
      },
      timeout: benefitGrantExecutionTimeoutMs
    }
  )
  return data
}

export async function createAutomatic(
  request: AutomaticBenefitGrantRequest
): Promise<BenefitGrantCampaign> {
  const { data } = await apiClient.post<BenefitGrantCampaign>(
    '/admin/benefit-grants/automatic',
    request,
    {
      headers: {
        'Idempotency-Key': request.operation_key
      },
      timeout: benefitGrantExecutionTimeoutMs
    }
  )
  return data
}

export async function list(
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<BenefitGrantCampaign>> {
  const { data } = await apiClient.get<PaginatedResponse<BenefitGrantCampaign>>(
    '/admin/benefit-grants',
    { params: { page, page_size: pageSize } }
  )
  return data
}

export async function get(id: number): Promise<BenefitGrantCampaign> {
  const { data } = await apiClient.get<BenefitGrantCampaign>(`/admin/benefit-grants/${id}`)
  return data
}

export async function listRecipients(
  campaignId: number,
  page: number = 1,
  pageSize: number = 50,
  status?: BenefitGrantRecipientStatus
): Promise<PaginatedResponse<BenefitGrantRecipient>> {
  const { data } = await apiClient.get<PaginatedResponse<BenefitGrantRecipient>>(
    `/admin/benefit-grants/${campaignId}/recipients`,
    {
      params: {
        page,
        page_size: pageSize,
        status: status || undefined
      }
    }
  )
  return data
}

export async function retry(id: number, idempotencyKey: string): Promise<BenefitGrantResult> {
  const { data } = await apiClient.post<BenefitGrantResult>(
    `/admin/benefit-grants/${id}/retry`,
    {},
    {
      headers: {
        'Idempotency-Key': idempotencyKey
      },
      timeout: benefitGrantExecutionTimeoutMs
    }
  )
  return data
}

export const benefitGrantsAPI = {
  preview,
  execute,
  createAutomatic,
  list,
  get,
  listRecipients,
  retry
}

export default benefitGrantsAPI
