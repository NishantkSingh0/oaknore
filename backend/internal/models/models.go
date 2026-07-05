package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ─── Organization ─────────────────────────────────────────

type Organization struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	Slug      string    `db:"slug"       json:"slug"`
	LogoURL   *string   `db:"logo_url"   json:"logo_url,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ─── Department ───────────────────────────────────────────

type DepartmentLayer string

const (
	LayerTwo   DepartmentLayer = "LAYER_2"
	LayerThree DepartmentLayer = "LAYER_3"
)

type Department struct {
	ID           uuid.UUID       `db:"id"            json:"id"`
	OrgID        uuid.UUID       `db:"org_id"        json:"org_id"`
	Name         string          `db:"name"          json:"name"`
	Layer        DepartmentLayer `db:"layer"         json:"layer"`
	ParentDeptID *uuid.UUID      `db:"parent_dept_id" json:"parent_dept_id,omitempty"`
	Description  *string         `db:"description"   json:"description,omitempty"`
	IsActive     bool            `db:"is_active"     json:"is_active"`
	CreatedAt    time.Time       `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"    json:"updated_at"`
}

// ─── User ─────────────────────────────────────────────────

type UserRole string

const (
	RoleSuperAdmin UserRole = "SUPER_ADMIN"
	RoleAdmin      UserRole = "ADMIN"
	RoleLayer2     UserRole = "LAYER_2"
	RoleLayer3     UserRole = "LAYER_3"
)

type User struct {
	ID           uuid.UUID  `db:"id"            json:"id"`
	OrgID        uuid.UUID  `db:"org_id"        json:"org_id"`
	EmployeeID   *string    `db:"employee_id"   json:"employee_id,omitempty"`
	FirstName    string     `db:"first_name"    json:"first_name"`
	LastName     string     `db:"last_name"     json:"last_name"`
	Email        string     `db:"email"         json:"email"`
	Phone        *string    `db:"phone"         json:"phone,omitempty"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Role         UserRole   `db:"role"          json:"role"`
	DepartmentID *uuid.UUID `db:"department_id" json:"department_id,omitempty"`
	IsActive     bool       `db:"is_active"     json:"is_active"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"    json:"updated_at"`
}

// ─── Project ──────────────────────────────────────────────

type ProjectStatus string

const (
	ProjectCreated    ProjectStatus = "CREATED"
	ProjectRouting    ProjectStatus = "ROUTING"
	ProjectInProgress ProjectStatus = "IN_PROGRESS"
	ProjectCompleted  ProjectStatus = "COMPLETED"
	ProjectArchived   ProjectStatus = "ARCHIVED"
)

type Project struct {
	ID              uuid.UUID     `db:"id"               json:"id"`
	OrgID           uuid.UUID     `db:"org_id"           json:"org_id"`
	PONumber        string        `db:"po_number"        json:"po_number"`
	ClientName      string        `db:"client_name"      json:"client_name"`
	ClientContact   *string       `db:"client_contact"   json:"client_contact,omitempty"`
	Name            string        `db:"name"             json:"name"`
	Quantity        int           `db:"quantity"         json:"quantity"`
	Dimensions      *string       `db:"dimensions"       json:"dimensions,omitempty"`
	Specifications  *string       `db:"specifications"   json:"specifications,omitempty"`
	MaterialDetails *string       `db:"material_details" json:"material_details,omitempty"`
	ColorDetails    *string       `db:"color_details"    json:"color_details,omitempty"`
	Upholstery      *string       `db:"upholstery"       json:"upholstery,omitempty"`
	Finish          *string       `db:"finish"           json:"finish,omitempty"`
	DeliveryDate    *time.Time    `db:"delivery_date"    json:"delivery_date,omitempty"`
	DeliveryAddress *string       `db:"delivery_address" json:"delivery_address,omitempty"`
	Remarks         *string       `db:"remarks"          json:"remarks,omitempty"`
	CoverImageURL   *string       `db:"cover_image_url"  json:"cover_image_url,omitempty"`
	CADFilesURL     *string       `db:"cad_files_url"    json:"cad_files_url,omitempty"`
	DrawingsURL     *string       `db:"drawings_url"     json:"drawings_url,omitempty"`
	JobCardsURL     *string       `db:"job_cards_url"    json:"job_cards_url,omitempty"`
	RenderFilesURL  *string       `db:"render_files_url" json:"render_files_url,omitempty"`
	Status          ProjectStatus `db:"status"           json:"status"`
	CurrentRevision int           `db:"current_revision" json:"current_revision"`
	CreatedBy       uuid.UUID     `db:"created_by"       json:"created_by"`
	CreatedAt       time.Time     `db:"created_at"       json:"created_at"`
	UpdatedAt       time.Time     `db:"updated_at"       json:"updated_at"`
}

// ─── Project Revision ─────────────────────────────────────

type ProjectRevision struct {
	ID                  uuid.UUID       `db:"id"                  json:"id"`
	ProjectID           uuid.UUID       `db:"project_id"          json:"project_id"`
	RevisionNumber      int             `db:"revision_number"     json:"revision_number"`
	UpdatedBy           uuid.UUID       `db:"updated_by"          json:"updated_by"`
	Reason              *string         `db:"reason"              json:"reason,omitempty"`
	ClientRequestRef    *string         `db:"client_request_ref"  json:"client_request_ref,omitempty"`
	PrevValues          []byte          `db:"prev_values"         json:"prev_values,omitempty"`
	NewValues           []byte          `db:"new_values"          json:"new_values,omitempty"`
	RoutingChanged      bool            `db:"routing_changed"     json:"routing_changed"`
	DepartmentsReopened pq.GenericArray `db:"departments_reopened" json:"departments_reopened,omitempty"`
	SubtasksReopened    pq.GenericArray `db:"subtasks_reopened"   json:"subtasks_reopened,omitempty"`
	NotificationsSent   bool            `db:"notifications_sent"  json:"notifications_sent"`
	CreatedAt           time.Time       `db:"created_at"          json:"created_at"`
}

// ─── File Asset ───────────────────────────────────────────

type FileOwnerType string

const (
	FileOwnerProject     FileOwnerType = "PROJECT"
	FileOwnerTask        FileOwnerType = "TASK"
	FileOwnerSubtask     FileOwnerType = "SUBTASK"
	FileOwnerIssue       FileOwnerType = "ISSUE"
	FileOwnerDailyReport FileOwnerType = "DAILY_REPORT"
	FileOwnerQuery       FileOwnerType = "QUERY"
	FileOwnerRevision    FileOwnerType = "REVISION"
	FileOwnerRework      FileOwnerType = "REWORK"
)

type FileAsset struct {
	ID         uuid.UUID     `db:"id"          json:"id"`
	OrgID      uuid.UUID     `db:"org_id"      json:"org_id"`
	ProjectID  *uuid.UUID    `db:"project_id"  json:"project_id,omitempty"`
	OwnerType  FileOwnerType `db:"owner_type"  json:"owner_type"`
	OwnerID    uuid.UUID     `db:"owner_id"    json:"owner_id"`
	FileName   string        `db:"file_name"   json:"file_name"`
	FileSize   *int64        `db:"file_size"   json:"file_size,omitempty"`
	MimeType   *string       `db:"mime_type"   json:"mime_type,omitempty"`
	S3Key      string        `db:"s3_key"      json:"s3_key"`
	URL        string        `db:"url"         json:"url"`
	UploadedBy uuid.UUID     `db:"uploaded_by" json:"uploaded_by"`
	CreatedAt  time.Time     `db:"created_at"  json:"created_at"`
}

// ─── Routing ──────────────────────────────────────────────

type RoutingStatus string

const (
	RoutingDraft      RoutingStatus = "DRAFT"
	RoutingActive     RoutingStatus = "ACTIVE"
	RoutingSuperseded RoutingStatus = "SUPERSEDED"
)

type DependencyPolicy string

const (
	RequireAll DependencyPolicy = "REQUIRE_ALL"
	RequireAny DependencyPolicy = "REQUIRE_ANY"
)

type Routing struct {
	ID              uuid.UUID     `db:"id"               json:"id"`
	ProjectID       uuid.UUID     `db:"project_id"       json:"project_id"`
	Version         int           `db:"version"          json:"version"`
	ParentRoutingID *uuid.UUID    `db:"parent_routing_id" json:"parent_routing_id,omitempty"`
	Status          RoutingStatus `db:"status"           json:"status"`
	CreatedBy       uuid.UUID     `db:"created_by"       json:"created_by"`
	Notes           *string       `db:"notes"            json:"notes,omitempty"`
	CreatedAt       time.Time     `db:"created_at"       json:"created_at"`
	UpdatedAt       time.Time     `db:"updated_at"       json:"updated_at"`
	Steps           []RoutingStep `db:"-"                json:"steps,omitempty"`
}

type RoutingStep struct {
	ID               uuid.UUID        `db:"id"                json:"id"`
	RoutingID        uuid.UUID        `db:"routing_id"        json:"routing_id"`
	StepOrder        int              `db:"step_order"        json:"step_order"`
	DependencyPolicy DependencyPolicy `db:"dependency_policy" json:"dependency_policy"`
	Label            *string          `db:"label"             json:"label,omitempty"`
	CreatedAt        time.Time        `db:"created_at"        json:"created_at"`
	DepartmentIDs    []uuid.UUID      `db:"-"                 json:"department_ids,omitempty"`
}

// ─── Department Task ──────────────────────────────────────

type TaskStatus string

const (
	TaskPending    TaskStatus = "PENDING"
	TaskInProgress TaskStatus = "IN_PROGRESS"
	TaskHold       TaskStatus = "HOLD"
	TaskIssueHold  TaskStatus = "ISSUE_HOLD"
	TaskCompleted  TaskStatus = "COMPLETED"
)

type DepartmentTask struct {
	ID            uuid.UUID  `db:"id"              json:"id"`
	ProjectID     uuid.UUID  `db:"project_id"      json:"project_id"`
	RoutingID     uuid.UUID  `db:"routing_id"      json:"routing_id"`
	RoutingStepID uuid.UUID  `db:"routing_step_id" json:"routing_step_id"`
	DepartmentID  uuid.UUID  `db:"department_id"   json:"department_id"`
	Status        TaskStatus `db:"status"          json:"status"`
	Priority      int        `db:"priority"        json:"priority"`
	StartDate     *time.Time `db:"start_date"      json:"start_date,omitempty"`
	DueDate       *time.Time `db:"due_date"        json:"due_date,omitempty"`
	DatesFrozen   bool       `db:"dates_frozen"    json:"dates_frozen"`
	Notes         *string    `db:"notes"           json:"notes,omitempty"`
	CreatedAt     time.Time  `db:"created_at"      json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"      json:"updated_at"`
	AssignedUsers []User     `db:"-"               json:"assigned_users,omitempty"`
	Subtasks      []Subtask  `db:"-"               json:"subtasks,omitempty"`
}

// ─── Subtask ──────────────────────────────────────────────

type SubtaskStatus string

const (
	SubtaskPending    SubtaskStatus = "PENDING"
	SubtaskInProgress SubtaskStatus = "IN_PROGRESS"
	SubtaskCompleted  SubtaskStatus = "COMPLETED"
)

type Subtask struct {
	ID          uuid.UUID     `db:"id"           json:"id"`
	TaskID      uuid.UUID     `db:"task_id"      json:"task_id"`
	ProjectID   uuid.UUID     `db:"project_id"   json:"project_id"`
	Title       string        `db:"title"        json:"title"`
	Description *string       `db:"description"  json:"description,omitempty"`
	IsRequired  bool          `db:"is_required"  json:"is_required"`
	Status      SubtaskStatus `db:"status"       json:"status"`
	AssignedTo  *uuid.UUID    `db:"assigned_to"  json:"assigned_to,omitempty"`
	CompletedAt *time.Time    `db:"completed_at" json:"completed_at,omitempty"`
	CompletedBy *uuid.UUID    `db:"completed_by" json:"completed_by,omitempty"`
	Notes       *string       `db:"notes"        json:"notes,omitempty"`
	SortOrder   int           `db:"sort_order"   json:"sort_order"`
	CreatedAt   time.Time     `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"   json:"updated_at"`
	Proofs      []FileAsset   `db:"-"            json:"proofs,omitempty"`
}

// ─── Issue ────────────────────────────────────────────────

type IssueType string

const (
	IssueMaterialMissing IssueType = "MATERIAL_MISSING"
	IssueDesignChange    IssueType = "DESIGN_CHANGE"
	IssueRoutingRequired IssueType = "ROUTING_REQUIRED"
	IssueFullScaleReq    IssueType = "FULL_SCALE_REQUIREMENT"
	IssueQualityIssue    IssueType = "QUALITY_ISSUE"
	IssueReworkRequired  IssueType = "REWORK_REQUIRED"
	IssueCustom          IssueType = "CUSTOM"
)

type IssueStatus string

const (
	IssueOpen            IssueStatus = "OPEN"
	IssuePendingApproval IssueStatus = "PENDING_APPROVAL"
	IssueApproved        IssueStatus = "APPROVED"
	IssueRejected        IssueStatus = "REJECTED"
	IssueResolved        IssueStatus = "RESOLVED"
	IssueClosed          IssueStatus = "CLOSED"
)

type Issue struct {
	ID             uuid.UUID   `db:"id"              json:"id"`
	ProjectID      uuid.UUID   `db:"project_id"      json:"project_id"`
	TaskID         *uuid.UUID  `db:"task_id"         json:"task_id,omitempty"`
	RaisedByDept   uuid.UUID   `db:"raised_by_dept"  json:"raised_by_dept"`
	RaisedBy       uuid.UUID   `db:"raised_by"       json:"raised_by"`
	IssueType      IssueType   `db:"issue_type"      json:"issue_type"`
	CustomType     *string     `db:"custom_type"     json:"custom_type,omitempty"`
	Title          string      `db:"title"           json:"title"`
	Description    string      `db:"description"     json:"description"`
	Status         IssueStatus `db:"status"          json:"status"`
	AssignedDept   *uuid.UUID  `db:"assigned_dept"   json:"assigned_dept,omitempty"`
	ReviewedBy     *uuid.UUID  `db:"reviewed_by"     json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time  `db:"reviewed_at"     json:"reviewed_at,omitempty"`
	ReviewNotes    *string     `db:"review_notes"    json:"review_notes,omitempty"`
	ResolvedBy     *uuid.UUID  `db:"resolved_by"     json:"resolved_by,omitempty"`
	ResolvedAt     *time.Time  `db:"resolved_at"     json:"resolved_at,omitempty"`
	ResolutionNote *string     `db:"resolution_note" json:"resolution_note,omitempty"`
	CreatedAt      time.Time   `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time   `db:"updated_at"      json:"updated_at"`
	Attachments    []FileAsset `db:"-"               json:"attachments,omitempty"`
}

// ─── Rework Request ───────────────────────────────────────

type ReworkStatus string

const (
	ReworkPending   ReworkStatus = "PENDING"
	ReworkApproved  ReworkStatus = "APPROVED"
	ReworkRejected  ReworkStatus = "REJECTED"
	ReworkCompleted ReworkStatus = "COMPLETED"
)

type ReworkRequest struct {
	ID                uuid.UUID    `db:"id"                   json:"id"`
	ProjectID         uuid.UUID    `db:"project_id"           json:"project_id"`
	OriginatingTaskID uuid.UUID    `db:"originating_task_id"  json:"originating_task_id"`
	RequestedBy       uuid.UUID    `db:"requested_by"         json:"requested_by"`
	RequestedDept     uuid.UUID    `db:"requested_dept"       json:"requested_dept"`
	TargetDeptID      uuid.UUID    `db:"target_dept_id"       json:"target_dept_id"`
	Reason            string       `db:"reason"               json:"reason"`
	Status            ReworkStatus `db:"status"               json:"status"`
	ReviewedBy        *uuid.UUID   `db:"reviewed_by"          json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time   `db:"reviewed_at"          json:"reviewed_at,omitempty"`
	ReviewNotes       *string      `db:"review_notes"         json:"review_notes,omitempty"`
	NewRoutingID      *uuid.UUID   `db:"new_routing_id"       json:"new_routing_id,omitempty"`
	CreatedAt         time.Time    `db:"created_at"           json:"created_at"`
	UpdatedAt         time.Time    `db:"updated_at"           json:"updated_at"`
}

// ─── Material Requisition ─────────────────────────────────

type MaterialReqStatus string

const (
	MatReqPending   MaterialReqStatus = "PENDING"
	MatReqApproved  MaterialReqStatus = "APPROVED"
	MatReqRejected  MaterialReqStatus = "REJECTED"
	MatReqFulfilled MaterialReqStatus = "FULFILLED"
)

type MaterialRequisition struct {
	ID          uuid.UUID         `db:"id"           json:"id"`
	ProjectID   uuid.UUID         `db:"project_id"   json:"project_id"`
	TaskID      *uuid.UUID        `db:"task_id"      json:"task_id,omitempty"`
	RequestedBy uuid.UUID         `db:"requested_by" json:"requested_by"`
	DeptID      uuid.UUID         `db:"dept_id"      json:"dept_id"`
	Status      MaterialReqStatus `db:"status"       json:"status"`
	Notes       *string           `db:"notes"        json:"notes,omitempty"`
	ReviewedBy  *uuid.UUID        `db:"reviewed_by"  json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time        `db:"reviewed_at"  json:"reviewed_at,omitempty"`
	ReviewNotes *string           `db:"review_notes" json:"review_notes,omitempty"`
	CreatedAt   time.Time         `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at"   json:"updated_at"`
	Items       []MaterialItem    `db:"-"            json:"items,omitempty"`
}

type MaterialItem struct {
	ID            uuid.UUID `db:"id"             json:"id"`
	RequisitionID uuid.UUID `db:"requisition_id" json:"requisition_id"`
	MaterialName  string    `db:"material_name"  json:"material_name"`
	Quantity      float64   `db:"quantity"       json:"quantity"`
	Unit          string    `db:"unit"           json:"unit"`
	Description   string    `db:"description"    json:"description"`
}

// ─── Query ────────────────────────────────────────────────

type QueryStatus string

const (
	QueryOpen             QueryStatus = "OPEN"
	QuerySenderResolved   QueryStatus = "SENDER_RESOLVED"
	QueryReceiverResolved QueryStatus = "RECEIVER_RESOLVED"
	QueryClosed           QueryStatus = "CLOSED"
)

type Query struct {
	ID               uuid.UUID      `db:"id"                json:"id"`
	ProjectID        uuid.UUID      `db:"project_id"        json:"project_id"`
	SenderID         uuid.UUID      `db:"sender_id"         json:"sender_id"`
	ReceiverID       uuid.UUID      `db:"receiver_id"       json:"receiver_id"`
	Subject          string         `db:"subject"           json:"subject"`
	Status           QueryStatus    `db:"status"            json:"status"`
	SenderResolved   bool           `db:"sender_resolved"   json:"sender_resolved"`
	ReceiverResolved bool           `db:"receiver_resolved" json:"receiver_resolved"`
	CreatedAt        time.Time      `db:"created_at"        json:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"        json:"updated_at"`
	Messages         []QueryMessage `db:"-"                 json:"messages,omitempty"`
}

type QueryMessage struct {
	ID        uuid.UUID   `db:"id"        json:"id"`
	QueryID   uuid.UUID   `db:"query_id"  json:"query_id"`
	SenderID  uuid.UUID   `db:"sender_id" json:"sender_id"`
	Body      string      `db:"body"      json:"body"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	Files     []FileAsset `db:"-"          json:"files,omitempty"`
}

// ─── Daily Report ─────────────────────────────────────────

type DailyReport struct {
	ID           uuid.UUID   `db:"id"            json:"id"`
	ProjectID    uuid.UUID   `db:"project_id"    json:"project_id"`
	DepartmentID uuid.UUID   `db:"department_id" json:"department_id"`
	SubmittedBy  uuid.UUID   `db:"submitted_by"  json:"submitted_by"`
	ReportDate   time.Time   `db:"report_date"   json:"report_date"`
	Description  string      `db:"description"   json:"description"`
	CreatedAt    time.Time   `db:"created_at"    json:"created_at"`
	Attachments  []FileAsset `db:"-"             json:"attachments,omitempty"`
}

// ─── Notification ─────────────────────────────────────────

type NotificationType string

const (
	NotifProjectCreated       NotificationType = "PROJECT_CREATED"
	NotifRoutingAssigned      NotificationType = "ROUTING_ASSIGNED"
	NotifRoutingUpdated       NotificationType = "ROUTING_UPDATED"
	NotifTaskAssigned         NotificationType = "TASK_ASSIGNED"
	NotifTaskStarted          NotificationType = "TASK_STARTED"
	NotifTaskCompleted        NotificationType = "TASK_COMPLETED"
	NotifSubtaskCompleted     NotificationType = "SUBTASK_COMPLETED"
	NotifProofUploaded        NotificationType = "PROOF_UPLOADED"
	NotifDailyReportSubmitted NotificationType = "DAILY_REPORT_SUBMITTED"
	NotifIssueRaised          NotificationType = "ISSUE_RAISED"
	NotifIssueApproved        NotificationType = "ISSUE_APPROVED"
	NotifIssueClosed          NotificationType = "ISSUE_CLOSED"
	NotifMaterialRequest      NotificationType = "MATERIAL_REQUEST"
	NotifReworkRequest        NotificationType = "REWORK_REQUEST"
	NotifQueryReceived        NotificationType = "QUERY_RECEIVED"
	NotifProjectRevision      NotificationType = "PROJECT_REVISION"
	NotifDeptReopened         NotificationType = "DEPARTMENT_REOPENED"
	NotifOverdueTask          NotificationType = "OVERDUE_TASK"
)

type Notification struct {
	ID            uuid.UUID        `db:"id"             json:"id"`
	OrgID         uuid.UUID        `db:"org_id"         json:"org_id"`
	RecipientID   uuid.UUID        `db:"recipient_id"   json:"recipient_id"`
	ProjectID     *uuid.UUID       `db:"project_id"     json:"project_id,omitempty"`
	Type          NotificationType `db:"type"           json:"type"`
	Title         string           `db:"title"          json:"title"`
	Body          *string          `db:"body"           json:"body,omitempty"`
	ReferenceID   *uuid.UUID       `db:"reference_id"   json:"reference_id,omitempty"`
	ReferenceType *string          `db:"reference_type" json:"reference_type,omitempty"`
	IsRead        bool             `db:"is_read"        json:"is_read"`
	CreatedAt     time.Time        `db:"created_at"     json:"created_at"`
}

// ─── Audit Log ────────────────────────────────────────────

type AuditLog struct {
	ID         uuid.UUID  `db:"id"          json:"id"`
	OrgID      uuid.UUID  `db:"org_id"      json:"org_id"`
	ProjectID  *uuid.UUID `db:"project_id"  json:"project_id,omitempty"`
	ActorID    *uuid.UUID `db:"actor_id"    json:"actor_id,omitempty"`
	Action     string     `db:"action"      json:"action"`
	EntityType string     `db:"entity_type" json:"entity_type"`
	EntityID   *uuid.UUID `db:"entity_id"   json:"entity_id,omitempty"`
	PrevState  []byte     `db:"prev_state"  json:"prev_state,omitempty"`
	NewState   []byte     `db:"new_state"   json:"new_state,omitempty"`
	Metadata   []byte     `db:"metadata"    json:"metadata,omitempty"`
	IPAddress  *string    `db:"ip_address"  json:"ip_address,omitempty"`
	CreatedAt  time.Time  `db:"created_at"  json:"created_at"`
}

// ─── Refresh Token ────────────────────────────────────────

type RefreshToken struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	UserID    uuid.UUID `db:"user_id"    json:"user_id"`
	TokenHash string    `db:"token_hash" json:"-"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	Revoked   bool      `db:"revoked"    json:"revoked"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
