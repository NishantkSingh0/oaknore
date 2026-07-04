import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'

const BASE_URL = import.meta.env.VITE_API_URL ?? '/api/v1'

export const api = axios.create({
  baseURL: BASE_URL,
  headers: { 'Content-Type': 'application/json' },
  timeout: 30_000,
})

// ── Inject access token ───────────────────────────────────
api.interceptors.request.use((cfg: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('access_token')
  if (token && cfg.headers) cfg.headers.Authorization = `Bearer ${token}`
  return cfg
})

// ── Auto-refresh on 401 ───────────────────────────────────
let isRefreshing = false
let queue: Array<(token: string) => void> = []

api.interceptors.response.use(
  (res) => res,
  async (err: AxiosError) => {
    const original = err.config as InternalAxiosRequestConfig & { _retry?: boolean }
    if (err.response?.status !== 401 || original._retry) {
      return Promise.reject(err)
    }
    original._retry = true

    if (isRefreshing) {
      return new Promise((resolve) => {
        queue.push((token) => {
          if (original.headers) original.headers.Authorization = `Bearer ${token}`
          resolve(api(original))
        })
      })
    }

    isRefreshing = true
    const refreshToken = localStorage.getItem('refresh_token')
    if (!refreshToken) {
      clearAuth()
      window.location.href = '/login'
      return Promise.reject(err)
    }

    try {
      const { data } = await axios.post(`${BASE_URL}/auth/refresh`, { refresh_token: refreshToken })
      const newToken: string = data.data.access_token
      localStorage.setItem('access_token', newToken)
      localStorage.setItem('refresh_token', data.data.refresh_token)
      queue.forEach((cb) => cb(newToken))
      queue = []
      if (original.headers) original.headers.Authorization = `Bearer ${newToken}`
      return api(original)
    } catch {
      clearAuth()
      window.location.href = '/login'
      return Promise.reject(err)
    } finally {
      isRefreshing = false
    }
  },
)

function clearAuth() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}
