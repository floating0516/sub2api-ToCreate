import { onMounted, onUnmounted, ref, type Ref } from 'vue'

export function useHtmlDarkMode(): Ref<boolean> {
  const isDark = ref(
    typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  )
  let observer: MutationObserver | null = null

  const sync = () => {
    isDark.value = document.documentElement.classList.contains('dark')
  }

  onMounted(() => {
    sync()
    observer = new MutationObserver(sync)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  })

  onUnmounted(() => {
    observer?.disconnect()
  })

  return isDark
}
