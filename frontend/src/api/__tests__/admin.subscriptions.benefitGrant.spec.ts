import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { post }
}))

import {
  executeBenefitGrant,
  previewBenefitGrant,
  type BenefitGrantRequest
} from '@/api/admin/subscriptions'

const request: BenefitGrantRequest = {
  audience_type: 'today_active',
  audience_date: '2026-07-30',
  timezone: 'Asia/Shanghai',
  benefit_type: 'subscription',
  group_id: 17,
  validity_days: 1,
  notes: 'summer campaign'
}

describe('admin benefit grant API', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('previews the resolved audience', async () => {
    post.mockResolvedValue({ data: { matched_count: 12, eligible_count: 10 } })

    const result = await previewBenefitGrant(request)

    expect(post).toHaveBeenCalledWith('/admin/benefit-grants/preview', request)
    expect(result.eligible_count).toBe(10)
  })

  it('executes with preview counts and an idempotency key', async () => {
    post.mockResolvedValue({ data: { granted_count: 10 } })

    const executeRequest = {
      ...request,
      expected_matched_count: 12,
      expected_eligible_count: 10
    }
    const result = await executeBenefitGrant(executeRequest, 'benefit-grant-operation-1')

    expect(post).toHaveBeenCalledWith(
      '/admin/benefit-grants/execute',
      executeRequest,
      {
        headers: {
          'Idempotency-Key': 'benefit-grant-operation-1'
        }
      }
    )
    expect(result.granted_count).toBe(10)
  })
})
