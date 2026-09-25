/**
 * Admin Subscriptions API endpoints
 * Handles user subscription management for administrators
 */

import { apiClient } from '../client'
import type {
  UserSubscription,
  SubscriptionProgress,
  AssignSubscriptionRequest,
  BulkAssignSubscriptionRequest,
  ExtendSubscriptionRequest,
  SubscriptionAddonPack,
  PaginatedResponse
} from '@/types'

export interface SubscriptionRetentionEstimate {
  estimated_retention_usd: number
  allocated_quota_usd: number
  used_quota_usd: number
  overage_usd: number
  subscription_count: number
  excluded_count: number
}

export type SubscriptionBulkAction = 'extend' | 'reset_quota' | 'revoke' | 'restore'

export interface SubscriptionBulkActionRequest {
  subscription_ids: number[]
  action: SubscriptionBulkAction
  days?: number
  daily?: boolean
  weekly?: boolean
  monthly?: boolean
}

export interface SubscriptionBulkActionResult {
  success_count: number
  failed_count: number
  results: Array<{ subscription_id: number; success: boolean; error?: string }>
}

export interface BulkAssignSubscriptionResult {
  success_count: number
  created_count: number
  reused_count: number
  failed_count: number
  subscriptions: UserSubscription[]
  errors: string[]
  statuses?: Record<string, 'created' | 'reused' | 'failed'>
}

export async function bulkAction(
  request: SubscriptionBulkActionRequest,
  idempotencyKey: string
): Promise<SubscriptionBulkActionResult> {
  const { data } = await apiClient.post<SubscriptionBulkActionResult>(
    '/admin/subscriptions/bulk-action',
    request,
    { headers: { 'Idempotency-Key': idempotencyKey } }
  )
  return data
}

/**
 * List all subscriptions with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters (status, user_id, group_id, sort_by, sort_order)
 * @returns Paginated list of subscriptions
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: 'active' | 'expired' | 'revoked' | 'suspended'
    user_id?: number
    group_id?: number
    platform?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    '/admin/subscriptions',
    {
      params: {
        page,
        page_size: pageSize,
        ...filters
      },
      signal: options?.signal
    }
  )
  return data
}

export async function getRetentionEstimate(filters?: {
  user_id?: number
  group_id?: number
  platform?: string
}, options?: { signal?: AbortSignal }): Promise<SubscriptionRetentionEstimate> {
  const { data } = await apiClient.get<SubscriptionRetentionEstimate>(
    '/admin/subscriptions/retention-estimate',
    { params: filters, signal: options?.signal }
  )
  return data
}

/**
 * Get subscription by ID
 * @param id - Subscription ID
 * @returns Subscription details
 */
export async function getById(id: number): Promise<UserSubscription> {
  const { data } = await apiClient.get<UserSubscription>(`/admin/subscriptions/${id}`)
  return data
}

/**
 * Get subscription progress
 * @param id - Subscription ID
 * @returns Subscription progress with usage stats
 */
export async function getProgress(id: number): Promise<SubscriptionProgress> {
  const { data } = await apiClient.get<SubscriptionProgress>(`/admin/subscriptions/${id}/progress`)
  return data
}

/**
 * Assign subscription to user
 * @param request - Assignment request
 * @returns Created subscription
 */
export async function assign(request: AssignSubscriptionRequest): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>('/admin/subscriptions/assign', request)
  return data
}

/**
 * Bulk assign subscriptions to multiple users
 * @param request - Bulk assignment request
 * @returns Per-user assignment outcomes and created or reused subscriptions
 */
export async function bulkAssign(
  request: BulkAssignSubscriptionRequest
): Promise<BulkAssignSubscriptionResult> {
  const { data } = await apiClient.post<BulkAssignSubscriptionResult>(
    '/admin/subscriptions/bulk-assign',
    request
  )
  return data
}

/**
 * Extend subscription validity
 * @param id - Subscription ID
 * @param request - Extension request with days
 * @returns Updated subscription
 */
export async function extend(
  id: number,
  request: ExtendSubscriptionRequest
): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(
    `/admin/subscriptions/${id}/extend`,
    request
  )
  return data
}

/**
 * Revoke subscription
 * @param id - Subscription ID
 * @returns Success confirmation
 */
export async function revoke(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/admin/subscriptions/${id}/revoke`)
  return data
}

/**
 * Restore revoked subscription
 * @param id - Subscription ID
 * @returns Restored subscription
 */
export async function restore(id: number): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(`/admin/subscriptions/${id}/restore`)
  return data
}

/**
 * Reset daily, weekly, and/or monthly usage quota for a subscription
 * @param id - Subscription ID
 * @param options - Which windows to reset
 * @returns Updated subscription
 */
export async function resetQuota(
  id: number,
  options: { daily: boolean; weekly: boolean; monthly: boolean }
): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(
    `/admin/subscriptions/${id}/reset-quota`,
    options
  )
  return data
}

export async function listAddons(id: number): Promise<SubscriptionAddonPack[]> {
  const { data } = await apiClient.get<SubscriptionAddonPack[]>(
    `/admin/subscriptions/${id}/addons`
  )
  return data
}

export async function grantAddon(
  id: number,
  request: { quota_usd: number; expires_at?: string; notes?: string }
): Promise<SubscriptionAddonPack> {
  const { data } = await apiClient.post<SubscriptionAddonPack>(
    `/admin/subscriptions/${id}/addons`,
    request
  )
  return data
}

export async function revokeAddon(id: number, addonId: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(
    `/admin/subscriptions/${id}/addons/${addonId}/revoke`
  )
  return data
}

/**
 * List subscriptions by group
 * @param groupId - Group ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of subscriptions in the group
 */
export async function listByGroup(
  groupId: number,
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    `/admin/groups/${groupId}/subscriptions`,
    {
      params: { page, page_size: pageSize }
    }
  )
  return data
}

/**
 * List subscriptions by user
 * @param userId - User ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of user's subscriptions
 */
export async function listByUser(
  userId: number,
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    `/admin/users/${userId}/subscriptions`,
    {
      params: { page, page_size: pageSize }
    }
  )
  return data
}

export const subscriptionsAPI = {
  list,
  getRetentionEstimate,
  getById,
  getProgress,
  assign,
  bulkAssign,
  bulkAction,
  extend,
  revoke,
  restore,
  resetQuota,
  listAddons,
  grantAddon,
  revokeAddon,
  listByGroup,
  listByUser
}

export default subscriptionsAPI
