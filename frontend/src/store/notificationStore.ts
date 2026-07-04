import { create } from 'zustand'
import type { Notification } from '@/types'

interface NotifState {
  notifications: Notification[]
  unreadCount: number
  setNotifications: (n: Notification[]) => void
  addNotification: (n: Notification) => void
  setUnreadCount: (c: number) => void
  decrementUnread: () => void
  markRead: (id: string) => void
  markAllRead: () => void
}

export const useNotifStore = create<NotifState>((set) => ({
  notifications: [],
  unreadCount: 0,
  setNotifications: (n) => set({ notifications: n }),
  addNotification: (n) =>
    set((s) => ({
      notifications: [n, ...s.notifications],
      unreadCount: s.unreadCount + 1,
    })),
  setUnreadCount: (c) => set({ unreadCount: c }),
  decrementUnread: () => set((s) => ({ unreadCount: Math.max(0, s.unreadCount - 1) })),
  markRead: (id) =>
    set((s) => ({
      notifications: s.notifications.map((n) =>
        n.id === id ? { ...n, is_read: true } : n,
      ),
    })),
  markAllRead: () =>
    set((s) => ({
      notifications: s.notifications.map((n) => ({ ...n, is_read: true })),
      unreadCount: 0,
    })),
}))
