// ─── Enums ────────────────────────────────────────────────

export type UserRole = 'SUPER_ADMIN' | 'ADMIN' | 'LAYER_2' | 'LAYER_3'
export type DepartmentLayer = 'LAYER_2' | 'LAYER_3'
export type ProjectStatus = 'CREATED' | 'ROUTING' | 'IN_PROGRESS' | 'COMPLETED' | 'ARCHIVED'
export type TaskStatus = 'PENDING' | 'IN_PROGRESS' | 'HOLD' | 'ISSUE_HOLD' | 'COMPLETED'
export type SubtaskStatus = 'PENDING' | 'IN_PROGRESS' | 'COMPLETED'
export type DependencyPolicy = 'REQUIRE_ALL' | 'REQUIRE_ANY'
export type RoutingStatus = 'DRAFT' | 'ACTIVE' | 'SUPERSEDED'
export type IssueType = 'MATERIAL_MISSING' | 'DESIGN_CHANGE' | 'ROUTING_REQUIRED' | 'FULL_SCALE_REQUIREMENT' | 'QUALITY_ISSUE' | 'REWORK_REQUIRED' | 'CUSTOM'
export type IssueStatus = 'OPEN' | 'PENDING_APPROVAL' | 'APPROVED' | 'REJECTED' | 'RESOLVED' | 'CLOSED'
export type ReworkStatus = 'PENDING' | 'APPROVED' | 'REJECTED' | 'COMPLETED'
export type MaterialReqStatus = 'PENDING' | 'APPROVED' | 'REJECTED' | 'FULFILLED'
export type QueryStatus = 'OPEN' | 'SENDER_RESOLVED' | 'RECEIVER_RESOLVED' | 'CLOSED'
export type NotificationType = 'PROJECT_CREATED' | 'ROUTING_ASSIGNED' | 'ROUTING_UPDATED' | 'TASK_ASSIGNED' | 'TASK_STARTED' | 'TASK_COMPLETED' | 'SUBTASK_COMPLETED' | 'PROOF_UPLOADED' | 'DAILY_REPORT_SUBMITTED' | 'ISSUE_RAISED' | 'ISSUE_APPROVED' | 'ISSUE_CLOSED' | 'MATERIAL_REQUEST' | 'REWORK_REQUEST' | 'QUERY_RECEIVED' | 'PROJECT_REVISION' | 'DEPARTMENT_REOPENED' | 'OVERDUE_TASK'

// ─── Entities ─────────────────────────────────────────────

export interface Organization { id: string; name: string; slug: string; logo_url?: string; created_at: string; updated_at: string }

export interface Department { id: string; org_id: string; name: string; layer: DepartmentLayer; parent_dept_id?: string; description?: string; is_active: boolean; created_at: string; updated_at: string }

export interface User { id: string; org_id: string; employee_id?: string; first_name: string; last_name: string; email: string; phone?: string; role: UserRole; department_id?: string; is_active: boolean; last_login_at?: string; created_at: string; updated_at: string }

export interface Project {
  id: string; org_id: string; po_number: string; client_name: string; client_contact?: string
  name: string; quantity: number; dimensions?: string; specifications?: string
  material_details?: string; color_details?: string; upholstery?: string; finish?: string
  delivery_date?: string; delivery_address?: string; remarks?: string
  cover_image_url?: string; cad_files_url?: string; drawings_url?: string
  job_cards_url?: string; render_files_url?: string
  status: ProjectStatus; current_revision: number; created_by: string
  created_at: string; updated_at: string
}

export interface ProjectRevision { id: string; project_id: string; revision_number: number; updated_by: string; reason?: string; client_request_ref?: string; prev_values?: object; new_values?: object; routing_changed: boolean; created_at: string }

export interface FileAsset { id: string; org_id: string; project_id?: string; owner_type: string; owner_id: string; file_name: string; file_size?: number; mime_type?: string; s3_key: string; url: string; uploaded_by: string; created_at: string }

export interface RoutingStep { id: string; routing_id: string; step_order: number; dependency_policy: DependencyPolicy; label?: string; department_ids: string[]; created_at: string }

export interface Routing { id: string; project_id: string; version: number; parent_routing_id?: string; status: RoutingStatus; created_by: string; notes?: string; steps: RoutingStep[]; created_at: string; updated_at: string }

export interface DepartmentTask { id: string; project_id: string; routing_id: string; routing_step_id: string; department_id: string; status: TaskStatus; priority: number; start_date?: string; due_date?: string; dates_frozen: boolean; notes?: string; assigned_users?: User[]; subtasks?: Subtask[]; created_at: string; updated_at: string }

export interface Subtask { id: string; task_id: string; project_id: string; title: string; description?: string; is_required: boolean; status: SubtaskStatus; assigned_to?: string; completed_at?: string; completed_by?: string; notes?: string; sort_order: number; proofs?: FileAsset[]; created_at: string; updated_at: string }

export interface Issue { id: string; project_id: string; task_id?: string; raised_by_dept: string; raised_by: string; issue_type: IssueType; custom_type?: string; title: string; description: string; status: IssueStatus; assigned_dept?: string; reviewed_by?: string; reviewed_at?: string; review_notes?: string; resolved_by?: string; resolved_at?: string; resolution_note?: string; attachments?: FileAsset[]; created_at: string; updated_at: string }

export interface ReworkRequest { id: string; project_id: string; originating_task_id: string; requested_by: string; requested_dept: string; target_dept_id: string; reason: string; status: ReworkStatus; reviewed_by?: string; reviewed_at?: string; review_notes?: string; new_routing_id?: string; created_at: string; updated_at: string }

export interface MaterialItem { id: string; requisition_id: string; material_name: string; quantity: number; unit: string; description: string }
export interface MaterialRequisition { id: string; project_id: string; task_id?: string; requested_by: string; dept_id: string; status: MaterialReqStatus; notes?: string; items: MaterialItem[]; reviewed_by?: string; reviewed_at?: string; created_at: string; updated_at: string }

export interface QueryMessage { id: string; query_id: string; sender_id: string; body: string; created_at: string }
export interface Query { id: string; project_id: string; sender_id: string; receiver_id: string; subject: string; status: QueryStatus; sender_resolved: boolean; receiver_resolved: boolean; messages?: QueryMessage[]; created_at: string; updated_at: string }

export interface DailyReport { id: string; project_id: string; department_id: string; submitted_by: string; report_date: string; description: string; attachments?: FileAsset[]; created_at: string }

export interface Notification { id: string; org_id: string; recipient_id: string; project_id?: string; type: NotificationType; title: string; body?: string; reference_id?: string; reference_type?: string; is_read: boolean; created_at: string }

export interface AuditLog { id: string; org_id: string; project_id?: string; actor_id?: string; action: string; entity_type: string; entity_id?: string; prev_state?: object; new_state?: object; ip_address?: string; created_at: string }

// ─── API response wrappers ────────────────────────────────

export interface ApiResponse<T> { success: boolean; data?: T; error?: string; message?: string }
export interface PaginatedResponse<T> { success: boolean; data: T[]; meta: { page: number; limit: number; total: number; total_pages: number } }

// ─── Auth ─────────────────────────────────────────────────

export interface AuthUser { id: string; email: string; first_name: string; last_name: string; role: UserRole; department_id?: string; org_id: string }
export interface LoginResponse { access_token: string; refresh_token: string; expires_in: number; user: AuthUser }
