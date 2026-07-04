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
    <aside
      className=" group flex h-screen w-[72px] hover:w-64 flex-col bg-gray-950 text-white transition-all duration-300 ease-in-out overflow-hidden"
    >      {/* Logo */}
      <div className="flex items-center gap-3 border-b border-white/10 px-5 py-5">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-brand-500 font-bold text-sm">
          P
        </div>

        <div
          className="
            opacity-0
            group-hover:opacity-100
            transition-opacity
            duration-200
            whitespace-nowrap
          "
        >
          <p className="text-sm font-semibold leading-none">PMS3</p>
          <p className="mt-0.5 text-[10px] text-brand-400">
            Production System
          </p>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-1">
        {nav
          .filter(n => n.roles.some(r => canAccess(r as any)))
          .map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                clsx(
                  "flex items-center rounded-lg px-3 py-2.5 transition-all",
                  isActive
                    ? "bg-gray-800 text-white"
                    : "text-brand-200 hover:bg-gray-800 hover:text-white"
                )
              }
            >
              <Icon
                size={18}
                className="shrink-0 mx-auto group-hover:mx-0 transition-all duration-300"
              />

              <span
                className="
                  ml-3
                  flex-1
                  overflow-hidden
                  whitespace-nowrap
                  opacity-0
                  w-0
                  group-hover:w-auto
                  group-hover:opacity-100
                  transition-all
                  duration-200
                "
              >
                {label}
              </span>

              {to === "/notifications" && unread > 0 && (
                <span
                  className="
                    ml-auto
                    opacity-0
                    group-hover:opacity-100
                    transition-opacity
                    flex h-4 min-w-4 items-center justify-center
                    rounded-full bg-red-500 px-1 text-[10px]
                  "
                >
                  {unread > 99 ? "99+" : unread}
                </span>
              )}
            </NavLink>
          ))}
      </nav>

      {/* User */}
      <div className="border-t border-white/10 p-3">
        <div className="flex items-center gap-3 rounded-lg px-3 py-2">

          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-600 text-xs font-semibold uppercase">
            {user?.first_name?.[0]}
            {user?.last_name?.[0]}
          </div>

          <div
            className="
              flex-1
              overflow-hidden
              opacity-0
              group-hover:opacity-100
              transition-opacity
            "
          >
            <p className="truncate text-xs font-medium">
              {user?.first_name} {user?.last_name}
            </p>

            <p className="truncate text-[10px] text-brand-400">
              {user?.role?.replace("_", " ")}
            </p>
          </div>

          <ChevronRight
            size={14}
            className="
              opacity-0
              group-hover:opacity-100
              transition-opacity
            "
          />
        </div>

        <button
          onClick={handleLogout}
          className="
            mt-2
            flex
            w-full
            items-center
            rounded-lg
            px-3
            py-2
            hover:bg-white/5
          "
        >
          <LogOut
            size={16}
            className="shrink-0 mx-auto group-hover:mx-0 transition-all"
          />

          <span
            className="
              ml-3
              opacity-0
              w-0
              overflow-hidden
              whitespace-nowrap
              group-hover:w-auto
              group-hover:opacity-100
              transition-all
            "
          >
            Sign out
          </span>
        </button>
      </div>
    </aside>
  )
}
