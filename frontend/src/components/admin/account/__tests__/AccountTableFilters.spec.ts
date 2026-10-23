import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import AccountTableFilters from '../AccountTableFilters.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
describe('account group multi-select', () => {
  it('selects multiple groups, includes ungrouped, searches and clears', async () => {
    const wrapper = mount(AccountTableFilters, {
      props: { searchQuery: '', filters: { group: '7' }, groups: [{ id: 7, name: 'Haiku' }, { id: 9, name: 'Sonnet' }] as any },
      global: { stubs: { Select: true, SearchInput: true } }
    })
    expect(wrapper.findAll('input[type="checkbox"]')[1].element).toHaveProperty('checked', true)
    await wrapper.findAll('input[type="checkbox"]')[2].setValue(true)
    expect(wrapper.emitted('update:filters')!.at(-1)).toEqual([{ group: '7,9' }])
    await wrapper.setProps({ filters: { group: '7,9' } })
    await wrapper.findAll('input[type="checkbox"]')[0].setValue(true)
    expect(wrapper.emitted('update:filters')!.at(-1)).toEqual([{ group: '7,9,ungrouped' }])
    await wrapper.find('input:not([type="checkbox"])').setValue('son')
    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(1)
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('update:filters')!.at(-1)).toEqual([{ group: '' }])
    wrapper.unmount()
  })
})
