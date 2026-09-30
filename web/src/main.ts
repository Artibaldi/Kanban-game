import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './ui/tokens.css'
import './ui/base.css'
import App from './app/App.vue'
import { router } from './app/router'
import { applyTheme, loadTheme } from './ui/theme'
import { setUnauthorizedHandler } from './api'
import { useSession } from './stores/session'

applyTheme(loadTheme())
const app = createApp(App)
app.use(createPinia())
app.use(router)
setUnauthorizedHandler(() => {
  useSession().me = null
  if (router.currentRoute.value.path !== '/login') router.push({ path: '/login', query: { expired: '1' } })
})
app.mount('#app')
