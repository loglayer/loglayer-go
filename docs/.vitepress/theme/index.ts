import type { Theme } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import ModuleBadges from './components/ModuleBadges.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('ModuleBadges', ModuleBadges)
  },
} satisfies Theme
