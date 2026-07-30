import { onMounted, onUnmounted, ref, type Ref } from 'vue'

/**
 * Shared open/close-on-outside-click-or-Escape behavior for small popover
 * UI (NotificationCenter, UserMenu today). Extracted rather than
 * duplicated in each component per CLAUDE.md's "Never duplicate business
 * logic" -- this is presentation-adjacent interaction logic, but the
 * same principle applies to any behavior two components would otherwise
 * copy verbatim.
 *
 * Each caller creates its own instance (not a module-level singleton
 * like useTheme/useSidebar): disclosure state is local to one popover,
 * not shared application state.
 *
 * Takes the root element ref rather than creating one, so the caller
 * owns it via Vue 3.5's `useTemplateRef('root')` bound to `ref="root"`
 * in its own template -- a template ref has to be created in the
 * component that owns the template it targets.
 */
export function useDisclosure(root: Readonly<Ref<HTMLElement | null>>) {
  const open = ref(false)

  function toggle() {
    open.value = !open.value
  }

  function close() {
    open.value = false
  }

  function handlePointerDown(event: MouseEvent) {
    if (open.value && root.value && !root.value.contains(event.target as Node)) {
      close()
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      close()
    }
  }

  onMounted(() => {
    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    document.removeEventListener('mousedown', handlePointerDown)
    document.removeEventListener('keydown', handleKeydown)
  })

  return { open, toggle, close }
}
