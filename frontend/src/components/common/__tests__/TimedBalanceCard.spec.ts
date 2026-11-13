import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import TimedBalanceCard from '../TimedBalanceCard.vue'

describe('TimedBalanceCard', () => {
  afterEach(() => vi.useRealTimers())
  const balance = { id: 1, amount: 10, remaining: 2.82, status: 'active' as const, created_at: '2026-11-01T00:00:00Z', expires_at: '2026-11-14T00:00:00Z' }
  it('shows remaining amount, progress and days', () => {
    vi.useFakeTimers(); vi.setSystemTime(new Date('2026-11-12T00:00:00Z'))
    const wrapper = mount(TimedBalanceCard, { props: { balance } })
    expect(wrapper.text()).toContain('$2.82')
    expect(wrapper.text()).toContain('余量 28.2%')
    expect(wrapper.text()).toContain('剩余 2 天')
    expect(Number(wrapper.get('[role="progressbar"]').attributes('aria-valuenow'))).toBeCloseTo(28.2)
    wrapper.unmount()
  })
  it('does not display expired funds as usable', () => {
    vi.useFakeTimers(); vi.setSystemTime(new Date('2026-11-15T00:00:00Z'))
    const wrapper = mount(TimedBalanceCard, { props: { balance } })
    expect(wrapper.text()).toContain('已过期')
    expect(wrapper.text()).toContain('$0.00')
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('0')
    wrapper.unmount()
  })
  it('shows less than one day and updates when the grant expires', async () => {
    vi.useFakeTimers(); vi.setSystemTime(new Date('2026-11-13T23:59:50Z'))
    const wrapper = mount(TimedBalanceCard, { props: { balance } })
    expect(wrapper.text()).toContain('不足 1 天')
    await vi.advanceTimersByTimeAsync(30000)
    expect(wrapper.text()).toContain('已过期')
    wrapper.unmount()
  })
})
