import { computed, inject, onBeforeUnmount, onMounted, provide, ref, type InjectionKey, type Ref } from 'vue'
import type { IconName } from '@/components/base/BaseIcon.vue'

/**
 * The reusable behavior behind DetailWorkspace/ContentsNavigation/
 * SectionCard (docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail Workspace
 * Structure"; docs/11-COMPONENT-ARCHITECTURE.md, "Workspace
 * Architecture"). DetailWorkspace.vue calls `provideDetailWorkspace()`
 * once; every SectionCard and the auto-rendered ContentsNavigation call
 * `useDetailWorkspaceContext()` to read/drive the same shared state via
 * provide/inject -- no configuration array or parallel navigation
 * structure is ever written by hand.
 *
 * Deliberately generic: this file has no notion of Customers, Services,
 * Devices, or any other business object. A "section" is just an id,
 * title, and optional icon.
 */
export interface DetailWorkspaceSection {
  id: string
  title: string
  icon?: IconName
}

export interface DetailWorkspaceContext {
  sections: Readonly<Ref<readonly DetailWorkspaceSection[]>>
  activeSectionId: Readonly<Ref<string | null>>
  isCollapsed: (id: string) => boolean
  toggleCollapsed: (id: string) => void
  goToSection: (id: string) => void
  registerSection: (section: DetailWorkspaceSection, el: HTMLElement) => void
  unregisterSection: (id: string) => void
}

const injectionKey: InjectionKey<DetailWorkspaceContext> = Symbol('DetailWorkspace')

export function useDetailWorkspaceContext(): DetailWorkspaceContext {
  const context = inject(injectionKey)
  if (!context) {
    throw new Error(
      'This component must be used inside <DetailWorkspace> (components/workspace/DetailWorkspace.vue).',
    )
  }
  return context
}

/** "Active Services" -> "active-services". Falls back to "section" for a title with no alphanumeric characters. */
export function slugifySectionTitle(title: string): string {
  return (
    title
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '') || 'section'
  )
}

// Roughly --shell-topnav-height (56px) plus breathing room, used both as
// the IntersectionObserver's trigger line and reflected in section.vue's
// scroll-margin-top so scrollIntoView doesn't tuck a section under the
// sticky app top bar.
export const DETAIL_WORKSPACE_SCROLL_OFFSET = 88

function prefersReducedMotion(): boolean {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

// WorkspaceHost (components/workspace/WorkspaceHost.vue) is its own
// scrolling container (`overflow-y: auto`), not the browser viewport --
// the app shell never lets the outer page scroll. IntersectionObserver's
// default root is the viewport, which would simply never intersect
// anything here since the viewport itself never scrolls. Walking up to
// find the actual scrolling ancestor keeps this correct without
// DetailWorkspace needing to know it lives inside WorkspaceHost
// specifically -- it would work the same inside any scrollable
// container.
function findScrollableAncestor(el: HTMLElement | null): HTMLElement | null {
  let node = el?.parentElement ?? null
  while (node) {
    if (/(auto|scroll)/.test(window.getComputedStyle(node).overflowY)) {
      return node
    }
    node = node.parentElement
  }
  return null
}

/**
 * Called once by DetailWorkspace.vue. Owns the section registry,
 * scrollspy (IntersectionObserver), scroll-to-section with focus
 * management, URL fragment sync, and per-section collapse state --
 * everything the docs describe DetailWorkspace as "coordinating".
 *
 * `containerRef` is DetailWorkspace's own root element, used only to
 * locate the nearest scrollable ancestor for the IntersectionObserver.
 */
export function provideDetailWorkspace(
  containerRef: Readonly<Ref<HTMLElement | null>>,
): DetailWorkspaceContext {
  const sections = ref<DetailWorkspaceSection[]>([])
  const activeSectionId = ref<string | null>(null)
  const collapsedIds = ref(new Set<string>())
  const elements = new Map<string, HTMLElement>()
  const intersecting = new Set<string>()
  let observer: IntersectionObserver | null = null

  function recomputeActive() {
    const first = sections.value.find((section) => intersecting.has(section.id))
    activeSectionId.value = first?.id ?? sections.value[0]?.id ?? null
  }

  function isCollapsed(id: string) {
    return collapsedIds.value.has(id)
  }

  function toggleCollapsed(id: string) {
    const next = new Set(collapsedIds.value)
    if (next.has(id)) {
      next.delete(id)
    } else {
      next.add(id)
    }
    collapsedIds.value = next
  }

  function registerSection(section: DetailWorkspaceSection, el: HTMLElement) {
    elements.set(section.id, el)
    sections.value = [...sections.value, section]
    observer?.observe(el)
    if (activeSectionId.value === null) {
      activeSectionId.value = section.id
    }
  }

  function unregisterSection(id: string) {
    const el = elements.get(id)
    if (el) observer?.unobserve(el)
    elements.delete(id)
    intersecting.delete(id)
    sections.value = sections.value.filter((section) => section.id !== id)
    recomputeActive()
  }

  function goToSection(id: string) {
    const el = elements.get(id)
    if (!el) return

    if (collapsedIds.value.has(id)) {
      toggleCollapsed(id)
    }

    const reducedMotion = prefersReducedMotion()
    el.scrollIntoView({ behavior: reducedMotion ? 'auto' : 'smooth', block: 'start' })

    if (window.location.hash !== `#${id}`) {
      window.history.pushState(null, '', `#${id}`)
    }

    activeSectionId.value = id

    // Move focus after the scroll settles, not immediately -- otherwise
    // focus lands before the section has actually scrolled into view.
    // There is no cross-browser "scroll finished" callback for
    // scrollIntoView, so this is a pragmatic estimate of smooth-scroll
    // duration, skipped entirely when motion is reduced.
    window.setTimeout(() => el.focus({ preventScroll: true }), reducedMotion ? 0 : 500)
  }

  function applyInitialHash() {
    const hash = window.location.hash.replace('#', '')
    if (!hash) return
    const el = elements.get(hash)
    if (!el) return
    if (collapsedIds.value.has(hash)) toggleCollapsed(hash)
    el.scrollIntoView({ behavior: 'auto', block: 'start' })
    activeSectionId.value = hash
  }

  onMounted(() => {
    observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            intersecting.add(entry.target.id)
          } else {
            intersecting.delete(entry.target.id)
          }
        }
        recomputeActive()
      },
      {
        root: findScrollableAncestor(containerRef.value),
        // A thin trigger band near the top of the scroll container: a
        // section becomes "active" once its top crosses below the
        // sticky app bar, and stops being active once it is 70%
        // scrolled past. Child components mount (and register) before
        // this runs, so every section is already known here.
        rootMargin: `-${DETAIL_WORKSPACE_SCROLL_OFFSET}px 0px -70% 0px`,
        threshold: 0,
      },
    )
    for (const el of elements.values()) {
      observer.observe(el)
    }
    applyInitialHash()
  })

  onBeforeUnmount(() => {
    observer?.disconnect()
    observer = null
  })

  const context: DetailWorkspaceContext = {
    sections: computed(() => sections.value),
    activeSectionId: computed(() => activeSectionId.value),
    isCollapsed,
    toggleCollapsed,
    goToSection,
    registerSection,
    unregisterSection,
  }

  provide(injectionKey, context)
  return context
}
