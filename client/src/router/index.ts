import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  // 登录页 —— 顶级路由，不套导航栏
  {
    path: '/login',
    name: 'login',
    component: () => import('../components/Login.vue'),
    meta: { requiresAuth: false },
  },

  // 需要导航栏的页面 —— 嵌套在 DefaultLayout 下
  {
    path: '/',
    component: () => import('../views/DefaultLayout.vue'),
    redirect: '/home',
    children: [
      {
        path: 'home',
        name: 'home',
        component: () => import('../views/Home.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'card',
        name: 'card',
        component: () => import('../views/Card.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'giftbooks',
        name: 'giftbooks',
        component: () => import('../views/GiftBooks.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'giftbooks/:id',
        name: 'giftbook-detail',
        component: () => import('../views/GiftBookDetail.vue'),
        meta: { requiresAuth: true, hidden: true },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// 路由守卫：未登录用户访问受保护的路由时，重定向到登录页
router.beforeEach((to, from) => {
  const token = localStorage.getItem('token')

  // 目标页面不需要登录，直接放行
  if (to.meta.requiresAuth !== true) {
    // 已登录用户访问 /login → 跳首页
    if (token && to.path === '/login') {
      return('/home')
      
    }
   
    return
  }

  // 目标页面需要登录，但没有 token，重定向到登录页
  if (!token) {
    return{ path: '/login', query: { redirect: to.fullPath } }
    
  }

  // 目标页面需要登录，且有 token，放行
  return
})

export default router
