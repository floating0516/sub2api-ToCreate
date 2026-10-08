import { createApp } from 'vue'
import { createI18n } from 'vue-i18n'
import zh from '../src/i18n/locales/zh/landing'
import en from '../src/i18n/locales/en/landing'
import Preview from './Preview.vue'
import './preview.css'

// The independent entry reuses the project's messages without loading stores,
// auth, router guards or any production API clients.
const language = new URLSearchParams(location.search).get('lang') === 'en' ? 'en' : 'zh'
const i18n = createI18n({ legacy: false, locale: language, fallbackLocale: 'en', messages: { zh, en } })
document.documentElement.lang = language
createApp(Preview).use(i18n).mount('#app')
