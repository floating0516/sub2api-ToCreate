import { apiClient } from './client'

export interface ManagedRechargeProduct {
  id: number
  slug: string
  plan_type: 'plus' | 'pro'
  name: string
  description: string
  price: number
  active: boolean
  sort_order: number
  available_stock: number
  total_stock: number
  created_at: string
  updated_at: string
}

export interface ManagedRechargeCatalog {
  enabled: boolean
  balance: number
  products: ManagedRechargeProduct[]
  mock_mode: boolean
  mock_step_seconds?: number
}

export interface ManagedRechargeOrder {
  id: number
  order_no: string
  user_id: number
  user_email?: string
  username?: string
  product_id: number
  product_slug: string
  product_name: string
  cdk_masked?: string
  price: number
  status: string
  account_email: string
  upstream_status?: string
  queue_position?: number
  queue_total?: number
  progress?: string
  error_code?: string
  error_message?: string
  balance_before?: number
  balance_after?: number
  paid_at?: string
  submitted_at?: string
  completed_at?: string
  refunded_at?: string
  last_synced_at?: string
  created_at: string
  updated_at: string
}

export interface ManagedRechargeCDK {
  id: number
  product_id: number
  product_name: string
  code_masked: string
  status: string
  expires_at?: string
  reserved_order_id?: number
  created_at: string
  updated_at: string
}

export interface ManagedRechargeCDKVerification {
  id: number
  valid: boolean
  expected_plan_type: 'plus' | 'pro'
  actual_plan_type?: 'plus' | 'pro'
  plan_name?: string
  processing_mode?: string
  matches_product: boolean
  verification_scope: 'verify_only'
}

export interface ManagedRechargeProductInput {
  slug: string
  plan_type: 'plus' | 'pro'
  name: string
  description: string
  price: number
  active: boolean
  sort_order: number
}

export async function getManagedRechargeCatalog(): Promise<ManagedRechargeCatalog> {
  const { data } = await apiClient.get<ManagedRechargeCatalog>('/managed-recharge/catalog')
  return data
}

export async function createManagedRechargeOrder(
  productId: number,
  sessionJson: string,
  idempotencyKey: string,
): Promise<ManagedRechargeOrder> {
  const { data } = await apiClient.post<ManagedRechargeOrder>(
    '/managed-recharge/orders',
    { product_id: productId, session_json: sessionJson },
    { headers: { 'Idempotency-Key': idempotencyKey }, timeout: 180000 },
  )
  return data
}

export async function listManagedRechargeOrders(limit = 20): Promise<ManagedRechargeOrder[]> {
  const { data } = await apiClient.get<ManagedRechargeOrder[]>('/managed-recharge/orders', {
    params: { limit },
  })
  return data
}

export async function getManagedRechargeOrder(id: number): Promise<ManagedRechargeOrder> {
  const { data } = await apiClient.get<ManagedRechargeOrder>(`/managed-recharge/orders/${id}`)
  return data
}

export async function submitManagedRechargeReplacementSession(
  id: number,
  sessionJson: string,
): Promise<ManagedRechargeOrder> {
  const { data } = await apiClient.post<ManagedRechargeOrder>(`/managed-recharge/orders/${id}/session`, {
    session_json: sessionJson,
  }, { timeout: 120000 })
  return data
}

export async function adminListManagedRechargeProducts(): Promise<ManagedRechargeProduct[]> {
  const { data } = await apiClient.get<ManagedRechargeProduct[]>('/admin/managed-recharge/products')
  return data
}

export async function adminCreateManagedRechargeProduct(
  input: ManagedRechargeProductInput,
): Promise<ManagedRechargeProduct> {
  const { data } = await apiClient.post<ManagedRechargeProduct>('/admin/managed-recharge/products', input)
  return data
}

export async function adminUpdateManagedRechargeProduct(
  id: number,
  input: ManagedRechargeProductInput,
): Promise<ManagedRechargeProduct> {
  const { data } = await apiClient.put<ManagedRechargeProduct>(`/admin/managed-recharge/products/${id}`, input)
  return data
}

export async function adminImportManagedRechargeCDKs(input: {
  product_id: number
  codes: string[]
  expires_at?: string
}): Promise<{ imported: number; skipped: number }> {
  const { data } = await apiClient.post<{ imported: number; skipped: number }>(
    '/admin/managed-recharge/cdks/import',
    input,
  )
  return data
}

export async function adminListManagedRechargeCDKs(params: {
  product_id?: number
  status?: string
  limit?: number
} = {}): Promise<ManagedRechargeCDK[]> {
  const { data } = await apiClient.get<ManagedRechargeCDK[]>('/admin/managed-recharge/cdks', { params })
  return data
}

export async function adminSetManagedRechargeCDKStatus(id: number, status: string): Promise<void> {
  await apiClient.put(`/admin/managed-recharge/cdks/${id}/status`, { status })
}

export async function adminVerifyManagedRechargeCDK(id: number): Promise<ManagedRechargeCDKVerification> {
  const { data } = await apiClient.post<ManagedRechargeCDKVerification>(
    `/admin/managed-recharge/cdks/${id}/verify`,
    undefined,
    { timeout: 60000 },
  )
  return data
}

export async function adminMoveManagedRechargeCDK(id: number, productId: number): Promise<void> {
  await apiClient.put(`/admin/managed-recharge/cdks/${id}/product`, { product_id: productId })
}

export async function adminListManagedRechargeOrders(
  status = '',
  limit = 100,
): Promise<ManagedRechargeOrder[]> {
  const { data } = await apiClient.get<ManagedRechargeOrder[]>('/admin/managed-recharge/orders', {
    params: { status, limit },
  })
  return data
}

export async function adminSyncManagedRechargeOrder(id: number): Promise<ManagedRechargeOrder> {
  const { data } = await apiClient.post<ManagedRechargeOrder>(`/admin/managed-recharge/orders/${id}/sync`)
  return data
}

export async function adminRefundManagedRechargeOrder(id: number): Promise<ManagedRechargeOrder> {
  const { data } = await apiClient.post<ManagedRechargeOrder>(`/admin/managed-recharge/orders/${id}/refund`)
  return data
}
