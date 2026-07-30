import { ref, watchEffect } from 'vue'

/**
 * Theme is App-owned state (docs/11-COMPONENT-ARCHITECTURE.md, "State
 * Ownership" table) shared by every component, not a single view's
 * concern -- so the reactive state lives at module scope and this
 * composable exposes a handle to that one instance rather than creating
 * a new ref per caller.
 */
export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'palladium.theme'
const media = window.matchMedia('(prefers-color-scheme: dark)')

function storedTheme(): Theme | null {
  const value = localStorage.getItem(STORAGE_KEY)
  return value === 'light' || value === 'dark' ? value : null
}

const theme = ref<Theme>(storedTheme() ?? (media.matches ? 'dark' : 'light'))

// Follow the OS theme live until the operator explicitly picks one.
media.addEventListener('change', (event) => {
  if (storedTheme() === null) {
    theme.value = event.matches ? 'dark' : 'light'
  }
})

watchEffect(() => {
  document.documentElement.setAttribute('data-theme', theme.value)
})

export function useTheme() {
  function setTheme(next: Theme) {
    theme.value = next
    localStorage.setItem(STORAGE_KEY, next)
  }

  function toggleTheme() {
    setTheme(theme.value === 'dark' ? 'light' : 'dark')
  }

  return { theme, setTheme, toggleTheme }
}
