import { useNavigate } from 'react-router-dom'
import { Bell, Search } from 'lucide-react'
import { useAuth } from '@/context/AuthContext'
import { useNotifStore } from '@/store/notificationStore'

export default function TopBar() {
  const { user } = useAuth()
  const unread = useNotifStore((s) => s.unreadCount)
  const navigate = useNavigate()

  return (
    <header className="flex items-center justify-between border-b border-gray-200 bg-white px-6 py-3.5">
      <div className="flex items-center gap-3">
        <button
          onClick={() => navigate('/search')}
          className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-500 hover:bg-gray-100 transition-colors w-64"
        >
          <Search size={14} />
          <span>Search projects, PO, client…</span>
        </button>
      </div>

      <div className="flex items-center gap-3">
        <button
          onClick={() => navigate('/notifications')}
          className="relative rounded-lg p-2 text-gray-500 hover:bg-gray-100 transition-colors"
        >
          <Bell size={18} />
          {unread > 0 && (
            <span className="absolute right-1 top-1 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-red-500 text-[9px] font-bold text-white">
              {unread > 9 ? '9+' : unread}
            </span>
          )}
        </button>

      </div>
    </header>
  )
}
