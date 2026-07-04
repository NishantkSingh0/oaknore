import { NavLink, useNavigate } from 'react-router-dom'
import { clsx } from 'clsx'
import {
  LayoutDashboard, FolderKanban, Users, Building2,
  Bell, Search, LogOut, ChevronRight, Settings,
} from 'lucide-react'
import { useAuth } from '@/context/AuthContext'
import { useNotifStore } from '@/store/notificationStore'

const nav = [
  { to: '/dashboard',   label: 'Dashboard',   icon: LayoutDashboard, roles: ['SUPER_ADMIN','ADMIN','LAYER_2','LAYER_3'] },
  { to: '/projects',    label: 'Projects',     icon: FolderKanban,    roles: ['SUPER_ADMIN','ADMIN','LAYER_2','LAYER_3'] },
  { to: '/departments', label: 'Departments',  icon: Building2,       roles: ['SUPER_ADMIN','ADMIN'] },
  { to: '/employees',   label: 'Employees',    icon: Users,           roles: ['SUPER_ADMIN','ADMIN'] },
  { to: '/search',      label: 'Search',       icon: Search,          roles: ['SUPER_ADMIN','ADMIN','LAYER_2','LAYER_3'] },
  { to: '/notifications', label: 'Notifications', icon: Bell,         roles: ['SUPER_ADMIN','ADMIN','LAYER_2','LAYER_3'] },
]

export default function Sidebar() {
  const { user, logout, canAccess } = useAuth()
  const unread = useNotifStore((s) => s.unreadCount)
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <aside className="flex h-screen w-60 flex-col bg-brand-950 text-white">
      {/* Logo */}
      <div className="flex items-center gap-3 px-5 py-5 border-b border-white/10">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-500 font-bold text-sm">P</div>
        <div>
          <p className="text-sm font-semibold leading-none">PMS3</p>
          <p className="text-[10px] text-brand-400 mt-0.5">Production System</p>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-0.5">
        {nav.filter(n => n.roles.some(r => canAccess(r as any))).map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) => clsx(
              'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
              isActive
                ? 'bg-brand-700 text-white'
                : 'text-brand-300 hover:bg-white/5 hover:text-white',
            )}
          >
            <Icon size={17} />
            <span className="flex-1">{label}</span>
            {to === '/notifications' && unread > 0 && (
              <span className="flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
                {unread > 99 ? '99+' : unread}
              </span>
            )}
          </NavLink>
        ))}
      </nav>

      {/* User */}
      <div className="border-t border-white/10 p-3 space-y-0.5">
        <div className="flex items-center gap-3 rounded-lg px-3 py-2.5">
          <div className="flex h-7 w-7 items-center justify-center rounded-full bg-brand-600 text-xs font-semibold uppercase">
            {user?.first_name?.[0]}{user?.last_name?.[0]}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-xs font-medium text-white truncate">{user?.first_name} {user?.last_name}</p>
            <p className="text-[10px] text-brand-400 truncate">{user?.role?.replace('_', ' ')}</p>
          </div>
          <ChevronRight size={14} className="text-brand-500" />
        </div>
        <button onClick={handleLogout} className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm text-brand-400 hover:bg-white/5 hover:text-white transition-colors">
          <LogOut size={15} />
          Sign out
        </button>
      </div>
    </aside>
  )
}
