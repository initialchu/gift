import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import router from '@/router'
import axios from '@/axios'

export const useAuthStore = defineStore('auth', () => {
  // ========== State ==========
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<{ username: string; role: string } | null>(null)

  // ========== Getters ==========
  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  // ========== 工具函数：从 JWT 中解码 payload ==========
  function decodeToken(tokenStr: string): { username: string; role: string } | null {
    try {
      const parts = tokenStr.split('.')
      if (!parts[1]) return null
      const payloadJson = atob(parts[1])
      return JSON.parse(payloadJson)
    } catch {
      return null
    }
  }

  // ========== Actions ==========

  // 登录 —— 返回 true 成功 / false 失败
  async function login(username: string, password: string, captchaid: string, captcha: string): Promise<boolean> {
    //  调登录 API，拿到 token
     const res = await axios.post('/auth/login', { username, password, captcha_id: captchaid, captcha_answer: captcha })
     token.value = res.data.token
     localStorage.setItem('token', res.data.token)
     const payload = decodeToken(res.data.token)
     if (payload) user.value = { username: payload.username, role: payload.role }
     return true
  }

  // 退出登录
  function logout(): void {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    router.push('/login')
  }

  // 页面刷新时恢复登录态（构造时自动调用一次）
  function checkAuth(): void {
    const savedToken = localStorage.getItem('token')
    if (!savedToken) return

    token.value = savedToken
    const payload = decodeToken(savedToken)
    if (payload) {
      user.value = { username: payload.username, role: payload.role }
    }
  }

  // 应用启动时自动恢复
  checkAuth()

  return { token, user, isLoggedIn, isAdmin, login, logout, checkAuth }
})
