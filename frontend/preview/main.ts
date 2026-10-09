import { createApp } from 'vue'
import { createI18n } from 'vue-i18n'
import zh from '../src/i18n/locales/zh/landing'
import en from '../src/i18n/locales/en/landing'
import zhCommon from '../src/i18n/locales/zh/common'
import enCommon from '../src/i18n/locales/en/common'
import Preview from './Preview.vue'
import './preview.css'

// The independent entry reuses the project's messages without loading stores,
// auth, router guards or any production API clients.
const language = new URLSearchParams(location.search).get('lang') === 'en' ? 'en' : 'zh'
const i18n = createI18n({ legacy: false, locale: language, fallbackLocale: 'en', messages: { zh: { ...zhCommon, ...zh }, en: { ...enCommon, ...en } } })
document.documentElement.lang = language
createApp(Preview).use(i18n).mount('#app')
