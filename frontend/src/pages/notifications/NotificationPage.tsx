import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notifApi } from '@/api/endpoints'
import { useNotifStore } from '@/store/notificationStore'
import { useWebSocket } from '@/hooks/useWebSocket'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import { Bell, CheckCheck, ExternalLink } from 'lucide-react'
import { format, formatDistanceToNow } from 'date-fns'
import type { Notification } from '@/types'
import toast from 'react-hot-toast'

const NOTIF_ICON_COLOR: Record<string, string> = {
  PROJECT_CREATED: 'bg-green-100 text-green-600',
  ROUTING_ASSIGNED: 'bg-blue-100 text-blue-600',
  ROUTING_UPDATED: 'bg-blue-100 text-blue-600',
  TASK_ASSIGNED: 'bg-purple-100 text-purple-600',
  TASK_COMPLETED: 'bg-green-100 text-green-600',
  ISSUE_RAISED: 'bg-red-100 text-red-600',
  ISSUE_APPROVED: 'bg-green-100 text-green-600',
  ISSUE_CLOSED: 'bg-gray-100 text-gray-500',
  REWORK_REQUEST: 'bg-orange-100 text-orange-600',
  QUERY_RECEIVED: 'bg-indigo-100 text-indigo-600',
  PROJECT_REVISION: 'bg-yellow-100 text-yellow-600',
  DEPARTMENT_REOPENED: 'bg-orange-100 text-orange-600',
  OVERDUE_TASK: 'bg-red-100 text-red-600',
  DAILY_REPORT_SUBMITTED: 'bg-teal-100 text-teal-600',
}

function getProjectLink(notif: Notification): string | null {
  if (!notif.project_id) return null
  if (notif.reference_type === 'ISSUE') return `/issues/${notif.reference_id}`
  if (notif.reference_type === 'TASK') return `/tasks/${notif.reference_id}`
  if (notif.reference_type === 'REWORK') return `/projects/${notif.project_id}/reworks`
  if (notif.reference_type === 'QUERY') return `/projects/${notif.project_id}/queries`
  return `/projects/${notif.project_id}`
}

export default function NotificationPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { setNotifications, setUnreadCount, addNotification, markRead, markAllRead } = useNotifStore()

  const { data, isLoading } = useQuery({
    queryKey: ['notifications'],
    queryFn: () => notifApi.list({ limit: 50 }),
    onSuccess: (res) => {
      setNotifications(res.data?.data ?? [])
    },
  } as any)

  const { data: countRes } = useQuery({
    queryKey: ['notif-count'],
    queryFn: () => notifApi.unreadCount(),
    onSuccess: (res: any) => setUnreadCount(res.data?.data?.count ?? 0),
  } as any)

  // Live WebSocket notifications
  useWebSocket((type, payload) => {
    const notif: Notification = {
      id: crypto.randomUUID(),
      org_id: '',
      recipient_id: '',
      type: type as any,
      title: type.replace(/_/g, ' '),
      body: JSON.stringify(payload),
      is_read: false,
      created_at: new Date().toISOString(),
      project_id: (payload as any)?.project_id,
      reference_id: (payload as any)?.reference_id,
      reference_type: (payload as any)?.reference_type,
    }
    addNotification(notif)
    toast(notif.title, { icon: '🔔' })
    qc.invalidateQueries({ queryKey: ['notif-count'] })
  })

  const { mutate: markOne } = useMutation({
    mutationFn: (id: string) => notifApi.markRead(id),
    onSuccess: (_, id) => { markRead(id); qc.invalidateQueries({ queryKey: ['notif-count'] }) },
  })

  const { mutate: markAll, isPending: markingAll } = useMutation({
    mutationFn: () => notifApi.markAllRead(),
    onSuccess: () => { markAllRead(); qc.invalidateQueries({ queryKey: ['notif-count'] }); toast.success('All marked as read') },
  })

  const notifications: Notification[] = data?.data?.data ?? []
  const unread = notifications.filter(n => !n.is_read).length

  const handleClick = (notif: Notification) => {
    if (!notif.is_read) markOne(notif.id)
    const link = getProjectLink(notif)
    if (link) navigate(link)
  }

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="max-w-2xl space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="page-title">Notifications</h1>
          {unread > 0 && <p className="text-sm text-gray-500 mt-0.5">{unread} unread</p>}
        </div>
        {unread > 0 && (
          <button onClick={() => markAll()} disabled={markingAll} className="btn-secondary btn-sm gap-1.5">
            <CheckCheck size={14} /> Mark all read
          </button>
        )}
      </div>

      {notifications.length === 0 ? (
        <EmptyState icon={Bell} title="No notifications" description="You're all caught up." />
      ) : (
        <div className="card divide-y divide-gray-50">
          {notifications.map(n => {
            const colorCls = NOTIF_ICON_COLOR[n.type] ?? 'bg-gray-100 text-gray-500'
            const link = getProjectLink(n)
            return (
              <div
                key={n.id}
                onClick={() => handleClick(n)}
                className={`flex items-start gap-3 px-5 py-4 cursor-pointer hover:bg-gray-50 transition-colors ${!n.is_read ? 'bg-brand-50' : ''}`}
              >
                <div className={`mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full ${colorCls}`}>
                  <Bell size={14} />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between gap-2">
                    <p className={`text-sm ${!n.is_read ? 'font-semibold text-gray-900' : 'font-medium text-gray-700'}`}>
                      {n.title}
                    </p>
                    <time className="text-xs text-gray-400 whitespace-nowrap flex-shrink-0">
                      {formatDistanceToNow(new Date(n.created_at), { addSuffix: true })}
                    </time>
                  </div>
                  {n.body && (
                    <p className="text-xs text-gray-500 mt-0.5 line-clamp-2">{n.body}</p>
                  )}
                  {link && (
                    <span className="mt-1 inline-flex items-center gap-1 text-xs text-brand-600">
                      <ExternalLink size={11} /> View
                    </span>
                  )}
                </div>
                {!n.is_read && (
                  <div className="mt-2 h-2 w-2 flex-shrink-0 rounded-full bg-brand-500" />
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
