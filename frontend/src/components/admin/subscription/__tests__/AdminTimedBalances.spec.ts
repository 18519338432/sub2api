import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import AdminTimedBalances from '../AdminTimedBalances.vue'
const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get } }))
describe('admin timed balances', () => {
  beforeEach(() => { get.mockReset(); get.mockResolvedValue({ data: { items: [], total: 13 } }) })
  it('scopes the list to the selected user and resets pagination when that user changes', async () => {
    const wrapper = mount(AdminTimedBalances, { props: { userId: 7 } })
    await flushPromises()
    expect(get).toHaveBeenLastCalledWith('/admin/subscriptions/timed-balances', { params: { user_id: 7, page: 1, page_size: 6, status: undefined } })
    await wrapper.findAll('button').find(b => b.text() === '下一页')!.trigger('click'); await flushPromises()
    expect(get.mock.lastCall?.[1].params.page).toBe(2)
    await wrapper.setProps({ userId: 8 }); await flushPromises()
    expect(get.mock.lastCall?.[1].params).toMatchObject({ user_id: 8, page: 1 })
    wrapper.unmount()
  })
  it('sends a status filter', async () => {
    const wrapper = mount(AdminTimedBalances)
    await flushPromises(); await wrapper.get('select').setValue('expired'); await flushPromises()
    expect(get.mock.lastCall?.[1].params.status).toBe('expired')
    wrapper.unmount()
  })
  it('shows a retryable error instead of an empty successful list', async () => {
    get.mockRejectedValue(new Error('offline'))
    const wrapper = mount(AdminTimedBalances)
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('加载失败')
    wrapper.unmount()
  })
})
