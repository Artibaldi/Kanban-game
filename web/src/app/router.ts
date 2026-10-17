import { createRouter, createWebHistory } from 'vue-router'
import { useSession } from '../stores/session'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: () => import('../pages/LoginPage.vue'), meta: { public: true, title: 'Вход' } },
    { path: '/', component: () => import('../pages/HomePage.vue'), meta: { title: 'Главная' } },
    { path: '/board', component: () => import('../pages/BoardPage.vue'), meta: { title: 'Доска' } },
    { path: '/review', component: () => import('../pages/ReviewPage.vue'), meta: { title: 'Проверка', lead: true } },
    { path: '/city', component: () => import('../pages/CityPage.vue'), meta: { title: 'Город', game: true } },
    { path: '/army', component: () => import('../pages/ArmyPage.vue'), meta: { title: 'Казарма', game: true } },
    { path: '/fortress', component: () => import('../pages/FortressPage.vue'), meta: { title: 'Крепость', game: true } },
    { path: '/fortress/duels', component: () => import('../pages/DuelsPage.vue'), meta: { title: 'Дружеские бои', game: true } },
    { path: '/opponents', redirect: '/fortress/duels' },
    { path: '/battles/:id', component: () => import('../pages/BattlePage.vue'), meta: { title: 'Бой', game: true } },
    { path: '/progress', component: () => import('../pages/ProgressPage.vue'), meta: { title: 'Прогресс' } },
    { path: '/ratings', redirect: '/progress' },
    { path: '/settings', component: () => import('../pages/SettingsPage.vue'), meta: { title: 'Настройки' } },
    { path: '/admin', component: () => import('../pages/AdminPage.vue'), meta: { title: 'Администрирование', admin: true } },
    { path: '/:rest(.*)*', redirect: '/' },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async to => {
  const session = useSession()
  if (!session.loaded) await session.load()
  if (to.meta.public) return session.me ? '/' : true
  if (!session.me) return { path: '/login', query: to.fullPath !== '/' ? { next: to.fullPath } : undefined }
  if (to.meta.lead && !session.isLead) return '/'
  if (to.meta.admin && !session.isAdmin) return '/'
  return true
})

router.afterEach(to => { document.title = `${to.meta.title || ''} — Бастион задач` })
