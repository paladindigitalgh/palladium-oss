import { it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseDisclosure from './BaseDisclosure.vue'

it('renders the title and starts collapsed by default', () => {
  const wrapper = mount(BaseDisclosure, {
    props: { title: 'Acme Fiber' },
    slots: { default: 'Plan rows go here' },
  })

  expect(wrapper.find('.base-disclosure__title').text()).toBe('Acme Fiber')
  expect(wrapper.find('.base-disclosure__collapsible').classes()).toContain('base-disclosure__collapsible--collapsed')
})

it('starts open when defaultOpen is set', () => {
  const wrapper = mount(BaseDisclosure, { props: { title: 'Acme Fiber', defaultOpen: true } })

  expect(wrapper.find('.base-disclosure__collapsible').classes()).not.toContain(
    'base-disclosure__collapsible--collapsed',
  )
})

it('toggles open and closed when the header is clicked', async () => {
  const wrapper = mount(BaseDisclosure, { props: { title: 'Acme Fiber' } })
  const toggle = wrapper.find('.base-disclosure__toggle')

  await toggle.trigger('click')
  expect(wrapper.find('.base-disclosure__collapsible').classes()).not.toContain(
    'base-disclosure__collapsible--collapsed',
  )
  expect(toggle.attributes('aria-expanded')).toBe('true')

  await toggle.trigger('click')
  expect(wrapper.find('.base-disclosure__collapsible').classes()).toContain('base-disclosure__collapsible--collapsed')
  expect(toggle.attributes('aria-expanded')).toBe('false')
})

it('renders the default slot content', () => {
  const wrapper = mount(BaseDisclosure, {
    props: { title: 'Acme Fiber' },
    slots: { default: '<p class="plan-row">Residential 500 Mbps</p>' },
  })

  expect(wrapper.find('.plan-row').exists()).toBe(true)
})

it('renders the extra slot next to the title', () => {
  const wrapper = mount(BaseDisclosure, {
    props: { title: 'Acme Fiber' },
    slots: { extra: '<span class="status-tag">Active</span>' },
  })

  expect(wrapper.find('.status-tag').text()).toBe('Active')
})
