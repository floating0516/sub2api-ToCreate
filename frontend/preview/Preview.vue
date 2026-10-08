<template>
  <div class="preview-home">
    <header class="preview-header"><nav aria-label="ToCreate">
      <a class="preview-brand" href="https://api.lihe.chat/"><img :src="logo" alt="" /><span>ToCreate</span></a>
      <div class="preview-nav-end"><span class="preview-tag">{{ t('home.modelDemo.preview') }}</span><button type="button" class="preview-language" :aria-label="t('home.modelDemo.language')" @click="switchLocale">{{ locale === 'zh' ? 'EN' : '中文' }}</button><a class="preview-login" href="https://api.lihe.chat/login">{{ t('home.login') }}<Icon name="arrowRight" :stroke-width="1.6" aria-hidden="true" /></a></div>
    </nav></header>
    <main>
      <section class="preview-hero">
        <div class="preview-copy"><h1>ToCreate</h1><h2>{{ t('home.heroSubtitle') }}</h2><p>{{ t('home.heroDescription') }}</p><a class="preview-primary" href="https://api.lihe.chat/login">{{ t('home.login') }}<Icon name="arrowRight" :stroke-width="1.6" aria-hidden="true" /></a></div>
        <div class="preview-visual">
          <template v-if="baseline">
            <article class="preview-original"><p>{{ t('home.redesign.diagramTitle') }}</p><ol><li><span class="provider-claude">C</span>Claude</li><li><span class="provider-gpt">O</span>GPT</li><li><span class="provider-gemini">G</span>Gemini</li></ol></article>
          </template>
          <ModelGatewayDemo v-else-if="showDemo" />
        </div>
      </section>
      <section class="preview-following"><div><p>{{ t('home.intro.eyebrow') }}</p><h2>{{ t('home.intro.title') }}</h2></div><span>{{ t('home.intro.subtitle') }}</span></section>
      <div class="preview-features"><article v-for="(feature,index) in features" :key="feature.title"><div><Icon :name="feature.icon" :stroke-width="1.6" aria-hidden="true" /><small>0{{ index+1 }}</small></div><h3>{{ t(feature.title) }}</h3><p>{{ t(feature.description) }}</p></article></div>
    </main>
    <footer class="preview-footer"><span>© {{ new Date().getFullYear() }} ToCreate</span><div><a :href="comparisonHref">{{ t(baseline ? 'home.modelDemo.newVersion' : 'home.modelDemo.baseline') }}</a><button v-if="qa" type="button" data-action="mount-toggle" @click="showDemo = !showDemo">{{ t(showDemo ? 'home.modelDemo.unmount' : 'home.modelDemo.mount') }}</button><a href="https://api.lihe.chat/">{{ t('home.modelDemo.back') }} ↗</a></div></footer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../src/components/icons/Icon.vue'
import ModelGatewayDemo from '../src/components/home/ModelGatewayDemo.vue'
import logo from './logo.png'
const { t, locale } = useI18n()
const params = new URLSearchParams(location.search)
const baseline = params.has('baseline'), qa = params.has('qa')
const showDemo = ref(true)
const comparisonHref = computed(() => `?${baseline ? '' : 'baseline=1&'}lang=${locale.value}`)
function switchLocale() { locale.value = locale.value === 'zh' ? 'en' : 'zh'; document.documentElement.lang = locale.value; const query = new URLSearchParams(location.search); query.set('lang', locale.value); history.replaceState(null, '', `?${query}`) }
const features = [
  { icon: 'key' as const, title: 'home.features.unifiedGateway', description: 'home.features.unifiedGatewayDesc' },
  { icon: 'swap' as const, title: 'home.features.multiAccount', description: 'home.features.multiAccountDesc' },
  { icon: 'chart' as const, title: 'home.features.balanceQuota', description: 'home.features.balanceQuotaDesc' }
]
</script>
