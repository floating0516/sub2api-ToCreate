<template>
  <div class="preview-home">
    <header class="preview-header">
      <nav aria-label="ToCreate">
        <a class="preview-brand" href="https://api.lihe.chat/">
          <img :src="logo" alt="" /><span>ToCreate</span>
        </a>
        <div class="preview-nav-end">
          <button type="button" class="preview-language" :aria-label="t('home.simplePreview.language')" @click="switchLocale">
            {{ locale === 'zh' ? 'EN' : '中文' }}
          </button>
          <EntryLink variant="quiet" data-action="login">{{ t('home.login') }}</EntryLink>
        </div>
      </nav>
    </header>
    <main class="preview-main">
      <section class="preview-hero" aria-labelledby="brand-title">
        <h1 id="brand-title">ToCreate<span aria-hidden="true">.</span></h1>
        <EntryLink data-action="start">{{ t('home.simplePreview.start') }}</EntryLink>
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
import { useI18n } from 'vue-i18n'
import EntryLink from './EntryLink.vue'
import logo from './logo.png'

const { t, locale } = useI18n()
function switchLocale() {
  locale.value = locale.value === 'zh' ? 'en' : 'zh'
  document.documentElement.lang = locale.value
  const query = new URLSearchParams(location.search)
  query.set('lang', locale.value)
  history.replaceState(null, '', `?${query}`)
}
</script>
