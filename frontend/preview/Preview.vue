<template>
  <OriginalAuthPreview v-if="view !== 'home'" :mode="view" :home-href="viewHref('home')" @mode="navigate" @home="navigate('home')" />
  <div v-else class="preview-home">
    <header class="preview-header">
      <nav aria-label="ToCreate">
        <a class="preview-brand" :href="viewHref('home')" @click.prevent="navigate('home')">
          <img :src="logo" alt="" /><span>ToCreate</span>
        </a>
        <div class="preview-nav-end">
          <button type="button" class="preview-language" :aria-label="t('home.simplePreview.language')" @click="switchLocale">
            {{ locale === 'zh' ? 'EN' : '中文' }}
          </button>
          <EntryLink v-if="view === 'home'" variant="quiet" data-action="login" :href="viewHref('login')" @click="openLogin">{{ t('home.login') }}</EntryLink>
        </div>
      </nav>
    </header>
    <main class="preview-main" :class="{ 'preview-main--auth': view !== 'home' }">
      <section v-if="view === 'home'" class="preview-hero" aria-labelledby="brand-title">
        <h1 id="brand-title">ToCreate<span aria-hidden="true">.</span></h1>
        <EntryLink data-action="start" :href="viewHref('login')" @click="openLogin">{{ t('home.simplePreview.start') }}</EntryLink>
      </section>
    </main>
    <footer class="preview-footer">
      <span>© {{ new Date().getFullYear() }} ToCreate</span>
      <span>{{ t('home.simplePreview.preview') }}</span>
      <a href="https://api.lihe.chat/">{{ t('home.simplePreview.back') }} ↗</a>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import OriginalAuthPreview from './OriginalAuthPreview.vue'
import { useI18n } from 'vue-i18n'
import EntryLink from './EntryLink.vue'
import logo from './logo.png'

const { t, locale } = useI18n()
type View = 'home' | 'login' | 'register'
function readView(): View { const value = new URLSearchParams(location.search).get('view'); return value === 'login' || value === 'register' ? value : 'home' }
const view = ref<View>(readView())
function viewHref(value: View) { const query = new URLSearchParams(location.search); if (value === 'home') query.delete('view'); else query.set('view', value); return `?${query}` }
function navigate(value: View) {
  if (view.value === value) return
  const wasHome = view.value === 'home'
  view.value = value
  history.pushState(null, '', viewHref(value))
  if (wasHome || value === 'home') void nextTick(() => {
    const target = document.querySelector<HTMLElement>(value === 'home' ? '#brand-title' : '#email-auth-title')
    target?.setAttribute('tabindex', '-1'); target?.focus({ preventScroll: true })
  })
}
function openLogin(event: MouseEvent) {
  if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || event.button !== 0) return
  event.preventDefault(); navigate('login')
}
function syncView() { view.value = readView() }
onMounted(() => window.addEventListener('popstate', syncView))
onBeforeUnmount(() => window.removeEventListener('popstate', syncView))
function switchLocale() {
  locale.value = locale.value === 'zh' ? 'en' : 'zh'
  document.documentElement.lang = locale.value
  const query = new URLSearchParams(location.search)
  query.set('lang', locale.value)
  history.replaceState(null, '', `?${query}`)
}
</script>
