import { ref, watchEffect } from 'vue'

/**
 * Sidebar collapse state is Navigation state (App-owned, see
 * docs/11-COMPONENT-ARCHITECTURE.md's "State Ownership" table), shared
 * between AppSidebar and the shell layout that reserves space for it --
 * hence a module-scope singleton rather than per-component local state.
 *
 * "collapsed" (desktop icon-only rail) is persisted, since it is a
 * deliberate operator preference (docs/07-UI-ARCHITECTURE.md section 10,
 * "Persistent Context"). "mobileOpen" (small-screen off-canvas overlay)
 * is intentionally not persisted -- it is a transient, per-visit
 * disclosure state, not a preference.
 */

const STORAGE_KEY = 'palladium.sidebar.collapsed'

// Must match AppSidebar.vue's `@media (max-width: 960px)` breakpoint --
// there is no way to read a CSS breakpoint from JS, so this is the one
// place it is duplicated as a literal. isMobileViewport exists so
// AppSidebar can mark itself `inert` while off-canvas: without it, the
// sidebar's links stay keyboard-focusable even while translated off
// screen on small viewports.
const MOBILE_BREAKPOINT = '(max-width: 960px)'

function storedCollapsed(): boolean {
  return localStorage.getItem(STORAGE_KEY) === 'true'
}

const collapsed = ref(storedCollapsed())
const mobileOpen = ref(false)

const mobileMedia = window.matchMedia(MOBILE_BREAKPOINT)
const isMobileViewport = ref(mobileMedia.matches)
mobileMedia.addEventListener('change', (event) => {
  isMobileViewport.value = event.matches
})

watchEffect(() => {
  localStorage.setItem(STORAGE_KEY, String(collapsed.value))
})

export function useSidebar() {
  function toggleCollapsed() {
    collapsed.value = !collapsed.value
  }

  function openMobile() {
    mobileOpen.value = true
  }

  function closeMobile() {
    mobileOpen.value = false
  }

  function toggleMobile() {
    mobileOpen.value = !mobileOpen.value
  }

  return {
    collapsed,
    toggleCollapsed,
    mobileOpen,
    openMobile,
    closeMobile,
    toggleMobile,
    isMobileViewport,
  }
}
