import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UserCheckinCalendar from '../UserCheckinCalendar.vue'

const { getCalendar, claim, refresh } = vi.hoisted(() => ({ getCalendar: vi.fn(), claim: vi.fn(), refresh: vi.fn() }))
vi.mock('@/api/activityRewards', () => ({ getCheckinCalendar: getCalendar, claimCheckin: claim }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ refreshUser: refresh }) }))
const calendar = {
  eligible: true,
  month: '2026-11', today: '2026-11-01', streak: 0, total_cents: 0,
  days: Array.from({ length: 30 }, (_, i) => ({ date: `2026-11-${String(i + 1).padStart(2, '0')}`, base_cents: 83, bonus_cents: 0, checked: false }))
}
describe('UserCheckinCalendar', () => {
  beforeEach(() => { vi.clearAllMocks(); getCalendar.mockResolvedValue(structuredClone(calendar)); refresh.mockResolvedValue(undefined) })
  it('uses all days of the natural month and disables claiming after success', async () => {
    const wrapper = mount(UserCheckinCalendar)
    await flushPromises()
    expect(wrapper.findAll('[aria-current="date"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('当月最高 $33')
    const received = structuredClone(calendar)
    received.days[0]!.checked = true; received.total_cents = 83; received.streak = 1
    claim.mockResolvedValue(received)
    await wrapper.get('button').trigger('click'); await flushPromises()
    expect(claim).toHaveBeenCalledTimes(1)
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('本月已领取 $0.83')
  })
  it('shows a retry action when loading fails', async () => {
    getCalendar.mockRejectedValueOnce(new Error('unavailable'))
    const wrapper = mount(UserCheckinCalendar); await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('重试')
    await wrapper.get('[role="alert"] button').trigger('click'); await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })
  it('blocks checkin for a user without an active subscription', async () => {
    getCalendar.mockResolvedValue({ ...structuredClone(calendar), eligible: false })
    const wrapper = mount(UserCheckinCalendar, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('仅限订阅用户')
    expect(claim).not.toHaveBeenCalled()
  })
})
