import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { authApi } from '@/api/endpoints'
import type { AuthUser } from '@/types'

interface AuthContextValue {
  user: AuthUser | null
  isLoading: boolean
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  canAccess: (...roles: AuthUser['role'][]) => boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  // Restore session on mount
  useEffect(() => {
    const token = localStorage.getItem('access_token')
    if (!token) { setIsLoading(false); return }
    authApi.me()
      .then((res) => setUser(res.data.data ?? null))
      .catch(() => {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
      })
      .finally(() => setIsLoading(false))
  }, [])

  const login = async (email: string, password: string) => {
    const res = await authApi.login(email, password)
    const payload = res.data.data!
    localStorage.setItem('access_token', payload.access_token)
    localStorage.setItem('refresh_token', payload.refresh_token)
    setUser(payload.user)
  }

  const logout = async () => {
    const rt = localStorage.getItem('refresh_token') ?? ''
    await authApi.logout(rt).catch(() => {})
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    setUser(null)
  }

  const canAccess = (...roles: AuthUser['role'][]) =>
    user ? roles.includes(user.role) : false

  return (
    <AuthContext.Provider value={{ user, isLoading, isAuthenticated: !!user, login, logout, canAccess }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
