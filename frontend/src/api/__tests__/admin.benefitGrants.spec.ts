import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post }
}))

import {
  execute,
  get as getCampaign,
  list as listCampaigns,
  listRecipients,
  preview,
  retry,
  type BenefitGrantRequest
} from '@/api/admin/benefitGrants'

const request: BenefitGrantRequest = {
  operation_key: 'benefit-grant-operation-1',
  audience_type: 'today_active',
  audience_date: '2026-07-30',
  audience_days: 1,
  timezone: 'Asia/Shanghai',
  benefit_type: 'subscription',
  conflict_policy: 'skip_active',
  group_id: 17,
  validity_days: 1,
  notes: 'summer campaign'
}

describe('admin benefit grants API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('previews the resolved audience', async () => {
    post.mockResolvedValue({ data: { matched_count: 12, eligible_count: 10 } })

    const result = await preview(request)

    expect(post).toHaveBeenCalledWith('/admin/benefit-grants/preview', request)
    expect(result.eligible_count).toBe(10)
  })

  it('executes with preview counts and an idempotency key', async () => {
    post.mockResolvedValue({ data: { granted_count: 10 } })

    const executeRequest = {
      ...request,
      expected_matched_count: 12,
      expected_eligible_count: 10,
      expected_snapshot: 'a'.repeat(64)
    }
    const result = await execute(executeRequest)

    expect(post).toHaveBeenCalledWith(
      '/admin/benefit-grants/execute',
      executeRequest,
      {
        headers: {
          'Idempotency-Key': 'benefit-grant-operation-1'
        },
        timeout: 90000
      }
    )
    expect(result.granted_count).toBe(10)
  })

  it('lists campaigns and recipients with pagination filters', async () => {
    get
      .mockResolvedValueOnce({ data: { items: [{ id: 11 }], total: 1 } })
      .mockResolvedValueOnce({ data: { items: [{ id: 21 }], total: 1 } })

    const campaigns = await listCampaigns(2, 25)
    const recipients = await listRecipients(11, 3, 50, 'failed')

    expect(get).toHaveBeenNthCalledWith(1, '/admin/benefit-grants', {
      params: { page: 2, page_size: 25 }
    })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/benefit-grants/11/recipients', {
      params: { page: 3, page_size: 50, status: 'failed' }
    })
    expect(campaigns.total).toBe(1)
    expect(recipients.total).toBe(1)
  })

  it('gets one campaign by ID', async () => {
    get.mockResolvedValue({ data: { id: 11, status: 'completed' } })

    const campaign = await getCampaign(11)

    expect(get).toHaveBeenCalledWith('/admin/benefit-grants/11')
    expect(campaign.status).toBe('completed')
  })

  it('retries an existing campaign with a separate idempotency key', async () => {
    post.mockResolvedValue({ data: { granted_count: 12, failed_count: 0 } })

    const result = await retry(11, 'benefit-grant-retry-11-operation-1')

    expect(post).toHaveBeenCalledWith(
      '/admin/benefit-grants/11/retry',
      {},
      {
        headers: {
          'Idempotency-Key': 'benefit-grant-retry-11-operation-1'
        },
        timeout: 90000
      }
    )
    expect(result.failed_count).toBe(0)
  })
})
