import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getTimedBalances } from '../subscriptions'
import { generate } from '../admin/redeem'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock('../client', () => ({ apiClient: { get, post } }))

describe('timed balances', () => {
  beforeEach(() => vi.clearAllMocks())

  it('keeps the grant lifetime separate from the code redemption deadline', async () => {
    post.mockResolvedValue({ data: [{ type: 'timed_balance', value: 12.5, validity_days: 7 }] })
    await generate(2, 'timed_balance', 12.5, undefined, 7, 30)
    expect(post).toHaveBeenCalledWith('/admin/redeem-codes/generate', {
      count: 2, type: 'timed_balance', value: 12.5,
      group_id: undefined, validity_days: 7, expires_in_days: 30
    })
  })

  it('loads the authenticated user grants including historical states', async () => {
    const grants = [{ id: 1, amount: 12.5, remaining: 0, status: 'exhausted' }]
    get.mockResolvedValue({ data: grants })
    expect(await getTimedBalances()).toEqual(grants)
    expect(get).toHaveBeenCalledWith('/subscriptions/timed-balances')
  })
})
