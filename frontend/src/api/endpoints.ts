import { api } from './client'
import type {
  LoginResponse, User, Organization, Department, Project, ProjectRevision,
  Routing, DepartmentTask, Subtask, Issue, ReworkRequest, MaterialRequisition,
  Query, QueryMessage, DailyReport, Notification, AuditLog, FileAsset,
  ApiResponse, PaginatedResponse,
} from '@/types'

// ── Auth ──────────────────────────────────────────────────
export const authApi = {
  login: (email: string, password: string) =>
    api.post<ApiResponse<LoginResponse>>('/auth/login', { email, password }),
  refresh: (refresh_token: string) =>
    api.post<ApiResponse<LoginResponse>>('/auth/refresh', { refresh_token }),
  logout: (refresh_token: string) =>
    api.post('/auth/logout', { refresh_token }),
  me: () => api.get<ApiResponse<User>>('/auth/me'),
  changePassword: (current_password: string, new_password: string) =>
    api.post('/auth/change-password', { current_password, new_password }),
}

// ── Organization ──────────────────────────────────────────
export const orgApi = {
  get: () => api.get<ApiResponse<Organization>>('/org'),
  update: (data: Partial<Organization>) => api.patch('/org', data),
}

// ── Departments ───────────────────────────────────────────
export const deptApi = {
  list: () => api.get<ApiResponse<Department[]>>('/departments'),
  get: (id: string) => api.get<ApiResponse<Department>>(`/departments/${id}`),
  create: (data: object) => api.post<ApiResponse<{ id: string }>>('/departments', data),
  update: (id: string, data: object) => api.patch(`/departments/${id}`, data),
}

// ── Employees ─────────────────────────────────────────────
export const empApi = {
  list: (params?: object) => api.get<PaginatedResponse<User>>('/employees', { params }),
  get: (id: string) => api.get<ApiResponse<User>>(`/employees/${id}`),
  create: (data: object) => api.post<ApiResponse<{ id: string }>>('/employees', data),
  update: (id: string, data: object) => api.patch(`/employees/${id}`, data),
  resetPassword: (id: string, new_password: string) =>
    api.post(`/employees/${id}/reset-password`, { new_password }),
  transfer: (id: string, department_id: string) =>
    api.patch(`/employees/${id}/transfer`, { department_id }),
}

// ── Projects ──────────────────────────────────────────────
export const projectApi = {
  list: (params?: object) => api.get<PaginatedResponse<Project>>('/projects', { params }),
  get: (id: string) => api.get<ApiResponse<Project>>(`/projects/${id}`),
  create: (data: object) => api.post<ApiResponse<{ id: string }>>('/projects', data),
  update: (id: string, data: object) => api.patch(`/projects/${id}`, data),
  updateStatus: (id: string, status: string) =>
    api.patch(`/projects/${id}/status`, { status }),
  revisions: (id: string) => api.get<ApiResponse<ProjectRevision[]>>(`/projects/${id}/revisions`),
  timeline: (id: string) => api.get<ApiResponse<AuditLog[]>>(`/projects/${id}/timeline`),
}

// ── Routing ───────────────────────────────────────────────
export const routingApi = {
  list: (projectId: string) =>
    api.get<ApiResponse<Routing[]>>(`/projects/${projectId}/routings`),
  get: (projectId: string, routingId: string) =>
    api.get<ApiResponse<Routing>>(`/projects/${projectId}/routings/${routingId}`),
  create: (projectId: string, data: object) =>
    api.post<ApiResponse<{ id: string; version: number }>>(`/projects/${projectId}/routings`, data),
}

// ── Tasks ─────────────────────────────────────────────────
export const taskApi = {
  list: (projectId: string, params?: object) =>
    api.get<ApiResponse<DepartmentTask[]>>(`/projects/${projectId}/tasks`, { params }),
  get: (id: string) => api.get<ApiResponse<DepartmentTask>>(`/tasks/${id}`),
  update: (id: string, data: object) => api.patch(`/tasks/${id}`, data),
  updateStatus: (id: string, status: string) =>
    api.patch(`/tasks/${id}/status`, { status }),
}

// ── Subtasks ──────────────────────────────────────────────
export const subtaskApi = {
  create: (taskId: string, data: object) =>
    api.post<ApiResponse<{ id: string }>>(`/tasks/${taskId}/subtasks`, data),
  update: (id: string, data: object) => api.patch(`/subtasks/${id}`, data),
  complete: (id: string) => api.patch(`/subtasks/${id}/complete`),
  delete: (id: string) => api.delete(`/subtasks/${id}`),
}

// ── Issues ────────────────────────────────────────────────
export const issueApi = {
  list: (projectId: string, params?: object) =>
    api.get<ApiResponse<Issue[]>>(`/projects/${projectId}/issues`, { params }),
  get: (id: string) => api.get<ApiResponse<Issue>>(`/issues/${id}`),
  create: (projectId: string, data: object) =>
    api.post<ApiResponse<{ id: string }>>(`/projects/${projectId}/issues`, data),
  review: (id: string, decision: string, notes?: string) =>
    api.patch(`/issues/${id}/review`, { decision, notes }),
  resolve: (id: string, resolution_note?: string) =>
    api.patch(`/issues/${id}/resolve`, { resolution_note }),
}

// ── Reworks ───────────────────────────────────────────────
export const reworkApi = {
  list: (projectId: string) =>
    api.get<ApiResponse<ReworkRequest[]>>(`/projects/${projectId}/reworks`),
  get: (id: string) => api.get<ApiResponse<ReworkRequest>>(`/reworks/${id}`),
  create: (projectId: string, data: object) =>
    api.post<ApiResponse<{ id: string }>>(`/projects/${projectId}/reworks`, data),
  review: (id: string, data: object) => api.patch(`/reworks/${id}/review`, data),
}

// ── Materials ─────────────────────────────────────────────
export const materialApi = {
  list: (projectId: string) =>
    api.get<ApiResponse<MaterialRequisition[]>>(`/projects/${projectId}/materials`),
  get: (id: string) => api.get<ApiResponse<MaterialRequisition>>(`/materials/${id}`),
  create: (projectId: string, data: object) =>
    api.post<ApiResponse<{ id: string }>>(`/projects/${projectId}/materials`, data),
  review: (id: string, decision: string, notes?: string) =>
    api.patch(`/materials/${id}/review`, { decision, notes }),
  fulfill: (id: string) => api.patch(`/materials/${id}/fulfill`),
}

// ── Queries ───────────────────────────────────────────────
export const queryApi = {
  list: (projectId: string) =>
    api.get<ApiResponse<Query[]>>(`/projects/${projectId}/queries`),
  get: (id: string) => api.get<ApiResponse<Query>>(`/queries/${id}`),
  create: (projectId: string, data: object) =>
    api.post<ApiResponse<{ id: string }>>(`/projects/${projectId}/queries`, data),
  postMessage: (id: string, body: string) =>
    api.post(`/queries/${id}/messages`, { body }),
  resolve: (id: string) => api.patch(`/queries/${id}/resolve`),
}

// ── Daily Reports ─────────────────────────────────────────
export const reportApi = {
  list: (projectId: string, params?: object) =>
    api.get<ApiResponse<DailyReport[]>>(`/projects/${projectId}/reports`, { params }),
  get: (id: string) => api.get<ApiResponse<DailyReport>>(`/reports/${id}`),
  create: (projectId: string, data: object) =>
    api.post<ApiResponse<{ id: string }>>(`/projects/${projectId}/reports`, data),
}

// ── Notifications ─────────────────────────────────────────
export const notifApi = {
  list: (params?: object) => api.get<PaginatedResponse<Notification>>('/notifications', { params }),
  unreadCount: () => api.get<ApiResponse<{ count: number }>>('/notifications/unread-count'),
  markRead: (id: string) => api.patch(`/notifications/${id}/read`),
  markAllRead: () => api.patch('/notifications/read-all'),
}

// ── Files ─────────────────────────────────────────────────
export const fileApi = {
  upload: (formData: FormData) =>
    api.post<ApiResponse<{ id: string; url: string; file_name: string }>>('/files/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  listByOwner: (owner_type: string, owner_id: string) =>
    api.get<ApiResponse<FileAsset[]>>('/files', { params: { owner_type, owner_id } }),
  presign: (id: string) => api.get<ApiResponse<{ url: string }>>(`/files/${id}/presign`),
  delete: (id: string) => api.delete(`/files/${id}`),
}

// ── Audit ─────────────────────────────────────────────────
export const auditApi = {
  list: (params?: object) => api.get<ApiResponse<AuditLog[]>>('/audit', { params }),
}
