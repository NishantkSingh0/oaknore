-- ─────────────────────────────────────────────────────────
--  PMS3 — Full Schema Migration (v1)
-- ─────────────────────────────────────────────────────────

-- Handy extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- for fast LIKE/ILIKE search

-- ─── ENUMS ───────────────────────────────────────────────

CREATE TYPE department_layer AS ENUM ('LAYER_2', 'LAYER_3');

CREATE TYPE user_role AS ENUM ('SUPER_ADMIN', 'ADMIN', 'LAYER_2', 'LAYER_3');

CREATE TYPE project_status AS ENUM (
    'CREATED', 'ROUTING', 'IN_PROGRESS', 'COMPLETED', 'ARCHIVED'
);

CREATE TYPE task_status AS ENUM (
    'PENDING', 'IN_PROGRESS', 'HOLD', 'ISSUE_HOLD', 'COMPLETED'
);

CREATE TYPE subtask_status AS ENUM ('PENDING', 'IN_PROGRESS', 'COMPLETED');

CREATE TYPE dependency_policy AS ENUM ('REQUIRE_ALL', 'REQUIRE_ANY');

CREATE TYPE routing_status AS ENUM ('DRAFT', 'ACTIVE', 'SUPERSEDED');

CREATE TYPE issue_type AS ENUM (
    'MATERIAL_MISSING', 'DESIGN_CHANGE', 'ROUTING_REQUIRED',
    'FULL_SCALE_REQUIREMENT', 'QUALITY_ISSUE', 'REWORK_REQUIRED', 'CUSTOM'
);

CREATE TYPE issue_status AS ENUM (
    'OPEN', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED', 'RESOLVED', 'CLOSED'
);

CREATE TYPE rework_status AS ENUM (
    'PENDING', 'APPROVED', 'REJECTED', 'COMPLETED'
);

CREATE TYPE material_req_status AS ENUM (
    'PENDING', 'APPROVED', 'REJECTED', 'FULFILLED'
);

CREATE TYPE query_status AS ENUM (
    'OPEN', 'SENDER_RESOLVED', 'RECEIVER_RESOLVED', 'CLOSED'
);

CREATE TYPE notification_type AS ENUM (
    'PROJECT_CREATED', 'ROUTING_ASSIGNED', 'ROUTING_UPDATED',
    'TASK_ASSIGNED', 'TASK_STARTED', 'TASK_COMPLETED',
    'SUBTASK_COMPLETED', 'PROOF_UPLOADED', 'DAILY_REPORT_SUBMITTED',
    'ISSUE_RAISED', 'ISSUE_APPROVED', 'ISSUE_CLOSED',
    'MATERIAL_REQUEST', 'REWORK_REQUEST', 'QUERY_RECEIVED',
    'PROJECT_REVISION', 'DEPARTMENT_REOPENED', 'OVERDUE_TASK'
);

CREATE TYPE file_owner_type AS ENUM (
    'PROJECT', 'TASK', 'SUBTASK', 'ISSUE', 'DAILY_REPORT',
    'QUERY', 'REVISION', 'REWORK'
);

-- ─── ORGANIZATION ─────────────────────────────────────────

CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    logo_url    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── DEPARTMENTS ─────────────────────────────────────────

CREATE TABLE departments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    layer           department_layer NOT NULL,
    parent_dept_id  UUID REFERENCES departments(id) ON DELETE SET NULL, -- Layer 3 under Layer 2
    description     TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_departments_org ON departments(org_id);
CREATE INDEX idx_departments_layer ON departments(layer);

-- ─── USERS / EMPLOYEES ───────────────────────────────────

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    employee_id     TEXT,                              -- company employee code
    first_name      TEXT NOT NULL,
    last_name       TEXT NOT NULL,
    email           TEXT NOT NULL UNIQUE,
    phone           TEXT,
    password_hash   TEXT NOT NULL,
    role            user_role NOT NULL DEFAULT 'LAYER_3',
    department_id   UUID REFERENCES departments(id) ON DELETE SET NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_org ON users(org_id);
CREATE INDEX idx_users_dept ON users(department_id);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_email ON users(email);

-- ─── PROJECTS ─────────────────────────────────────────────

CREATE TABLE projects (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    po_number           TEXT NOT NULL,
    client_name         TEXT NOT NULL,
    client_contact      TEXT,
    name                TEXT NOT NULL,
    quantity            INTEGER NOT NULL DEFAULT 1,
    dimensions          TEXT,
    specifications      TEXT,
    material_details    TEXT,
    color_details       TEXT,
    upholstery          TEXT,
    finish              TEXT,
    delivery_date       DATE,
    delivery_address    TEXT,
    remarks             TEXT,
    cover_image_url     TEXT,
    cad_files_url       TEXT,
    drawings_url        TEXT,
    job_cards_url       TEXT,
    render_files_url    TEXT,
    status              project_status NOT NULL DEFAULT 'CREATED',
    current_revision    INTEGER NOT NULL DEFAULT 0,
    created_by          UUID NOT NULL REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_org ON projects(org_id);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_po ON projects(po_number);
CREATE INDEX idx_projects_client ON projects(client_name);
CREATE INDEX idx_projects_created_by ON projects(created_by);
-- Full-text search index
CREATE INDEX idx_projects_fts ON projects USING gin(
    to_tsvector('english', coalesce(po_number,'') || ' ' || coalesce(client_name,'') || ' ' || coalesce(name,''))
);

-- ─── PROJECT REVISIONS ────────────────────────────────────

CREATE TABLE project_revisions (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id           UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    revision_number      INTEGER NOT NULL,
    updated_by           UUID NOT NULL REFERENCES users(id),
    reason               TEXT,
    client_request_ref   TEXT,
    prev_values          JSONB,          -- snapshot of previous field values
    new_values           JSONB,          -- snapshot of new field values
    routing_changed      BOOLEAN NOT NULL DEFAULT FALSE,
    departments_reopened UUID[],
    subtasks_reopened    UUID[],
    notifications_sent   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, revision_number)
);

CREATE INDEX idx_proj_revisions_project ON project_revisions(project_id);

-- ─── FILE ASSETS ──────────────────────────────────────────

CREATE TABLE file_assets (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id    UUID REFERENCES projects(id) ON DELETE CASCADE,
    owner_type    file_owner_type NOT NULL,
    owner_id      UUID NOT NULL,
    file_name     TEXT NOT NULL,
    file_size     BIGINT,
    mime_type     TEXT,
    s3_key        TEXT NOT NULL,
    url           TEXT NOT NULL,
    uploaded_by   UUID NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_file_assets_owner ON file_assets(owner_type, owner_id);
CREATE INDEX idx_file_assets_project ON file_assets(project_id);

-- ─── ROUTING ──────────────────────────────────────────────

CREATE TABLE routings (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id        UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version           INTEGER NOT NULL DEFAULT 1,
    parent_routing_id UUID REFERENCES routings(id) ON DELETE SET NULL, -- rework/revision lineage
    status            routing_status NOT NULL DEFAULT 'DRAFT',
    created_by        UUID NOT NULL REFERENCES users(id),
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, version)
);

CREATE INDEX idx_routings_project ON routings(project_id);
CREATE INDEX idx_routings_status ON routings(status);

-- ─── ROUTING STEPS ────────────────────────────────────────

CREATE TABLE routing_steps (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    routing_id        UUID NOT NULL REFERENCES routings(id) ON DELETE CASCADE,
    step_order        INTEGER NOT NULL,           -- 1-based sequential order
    dependency_policy dependency_policy NOT NULL DEFAULT 'REQUIRE_ALL',
    label             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_routing_steps_routing ON routing_steps(routing_id);

-- Junction: which departments belong to a routing step (parallel departments share a step)
CREATE TABLE routing_step_departments (
    routing_step_id UUID NOT NULL REFERENCES routing_steps(id) ON DELETE CASCADE,
    department_id   UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    PRIMARY KEY (routing_step_id, department_id)
);

-- ─── DEPARTMENT TASKS ─────────────────────────────────────

CREATE TABLE department_tasks (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    routing_id       UUID NOT NULL REFERENCES routings(id) ON DELETE CASCADE,
    routing_step_id  UUID NOT NULL REFERENCES routing_steps(id) ON DELETE CASCADE,
    department_id    UUID NOT NULL REFERENCES departments(id),
    status           task_status NOT NULL DEFAULT 'PENDING',
    priority         SMALLINT NOT NULL DEFAULT 2,  -- 1 Low, 2 Medium, 3 High
    start_date       DATE,
    due_date         DATE,
    dates_frozen     BOOLEAN NOT NULL DEFAULT FALSE,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dept_tasks_project ON department_tasks(project_id);
CREATE INDEX idx_dept_tasks_dept ON department_tasks(department_id);
CREATE INDEX idx_dept_tasks_status ON department_tasks(status);
CREATE INDEX idx_dept_tasks_routing ON department_tasks(routing_id);

-- Employees assigned to a task
CREATE TABLE task_assignments (
    task_id     UUID NOT NULL REFERENCES department_tasks(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, user_id)
);

-- ─── SUBTASKS ─────────────────────────────────────────────

CREATE TABLE subtasks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id         UUID NOT NULL REFERENCES department_tasks(id) ON DELETE CASCADE,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT,
    is_required     BOOLEAN NOT NULL DEFAULT TRUE,
    status          subtask_status NOT NULL DEFAULT 'PENDING',
    assigned_to     UUID REFERENCES users(id) ON DELETE SET NULL,
    completed_at    TIMESTAMPTZ,
    completed_by    UUID REFERENCES users(id),
    notes           TEXT,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subtasks_task ON subtasks(task_id);
CREATE INDEX idx_subtasks_project ON subtasks(project_id);

-- ─── ISSUES ───────────────────────────────────────────────

CREATE TABLE issues (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id         UUID REFERENCES department_tasks(id) ON DELETE SET NULL,
    raised_by_dept  UUID NOT NULL REFERENCES departments(id),
    raised_by       UUID NOT NULL REFERENCES users(id),
    issue_type      issue_type NOT NULL,
    custom_type     TEXT,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    status          issue_status NOT NULL DEFAULT 'OPEN',
    assigned_dept   UUID REFERENCES departments(id),
    reviewed_by     UUID REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ,
    review_notes    TEXT,
    resolved_by     UUID REFERENCES users(id),
    resolved_at     TIMESTAMPTZ,
    resolution_note TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_issues_project ON issues(project_id);
CREATE INDEX idx_issues_status ON issues(status);
CREATE INDEX idx_issues_task ON issues(task_id);

-- ─── REWORK REQUESTS ──────────────────────────────────────

CREATE TABLE rework_requests (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    originating_task_id UUID NOT NULL REFERENCES department_tasks(id),
    requested_by        UUID NOT NULL REFERENCES users(id),
    requested_dept      UUID NOT NULL REFERENCES departments(id),  -- dept to rework FROM
    target_dept_id      UUID NOT NULL REFERENCES departments(id),  -- dept to send work BACK to
    reason              TEXT NOT NULL,
    status              rework_status NOT NULL DEFAULT 'PENDING',
    reviewed_by         UUID REFERENCES users(id),
    reviewed_at         TIMESTAMPTZ,
    review_notes        TEXT,
    new_routing_id      UUID REFERENCES routings(id), -- routing created on approval
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rework_project ON rework_requests(project_id);
CREATE INDEX idx_rework_status ON rework_requests(status);

-- ─── MATERIAL REQUISITIONS ────────────────────────────────

CREATE TABLE material_requisitions (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id      UUID REFERENCES department_tasks(id),
    requested_by UUID NOT NULL REFERENCES users(id),
    dept_id      UUID NOT NULL REFERENCES departments(id),
    status       material_req_status NOT NULL DEFAULT 'PENDING',
    notes        TEXT,
    reviewed_by  UUID REFERENCES users(id),
    reviewed_at  TIMESTAMPTZ,
    review_notes TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE material_items (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    requisition_id   UUID NOT NULL REFERENCES material_requisitions(id) ON DELETE CASCADE,
    material_name    TEXT NOT NULL,
    quantity         NUMERIC(12,4) NOT NULL,
    unit             TEXT NOT NULL,
    description      TEXT
);

CREATE INDEX idx_mat_req_project ON material_requisitions(project_id);
CREATE INDEX idx_mat_req_status ON material_requisitions(status);

-- ─── QUERIES (CROSS-LAYER COMMUNICATION) ─────────────────

CREATE TABLE queries (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    sender_id     UUID NOT NULL REFERENCES users(id),
    receiver_id   UUID NOT NULL REFERENCES users(id),
    subject       TEXT NOT NULL,
    status        query_status NOT NULL DEFAULT 'OPEN',
    sender_resolved   BOOLEAN NOT NULL DEFAULT FALSE,
    receiver_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_queries_project ON queries(project_id);
CREATE INDEX idx_queries_sender ON queries(sender_id);
CREATE INDEX idx_queries_receiver ON queries(receiver_id);

CREATE TABLE query_messages (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    query_id   UUID NOT NULL REFERENCES queries(id) ON DELETE CASCADE,
    sender_id  UUID NOT NULL REFERENCES users(id),
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_query_msgs_query ON query_messages(query_id);

-- ─── DAILY REPORTS ────────────────────────────────────────

CREATE TABLE daily_reports (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id),
    submitted_by  UUID NOT NULL REFERENCES users(id),
    report_date   DATE NOT NULL,
    description   TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_daily_reports_project ON daily_reports(project_id);
CREATE INDEX idx_daily_reports_dept ON daily_reports(department_id);
CREATE INDEX idx_daily_reports_date ON daily_reports(report_date);

-- ─── NOTIFICATIONS ────────────────────────────────────────

CREATE TABLE notifications (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    recipient_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id    UUID REFERENCES projects(id) ON DELETE CASCADE,
    type          notification_type NOT NULL,
    title         TEXT NOT NULL,
    body          TEXT,
    reference_id  UUID,                 -- ID of the entity that triggered this notification
    reference_type TEXT,               -- e.g. 'TASK', 'ISSUE', 'REWORK'
    is_read       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient ON notifications(recipient_id, is_read);
CREATE INDEX idx_notifications_project ON notifications(project_id);

-- ─── AUDIT LOG ────────────────────────────────────────────
-- Append-only. No UPDATE/DELETE ever permitted on this table.

CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id    UUID REFERENCES projects(id) ON DELETE SET NULL,
    actor_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT NOT NULL,       -- e.g. 'PROJECT_CREATED', 'TASK_STATUS_CHANGED'
    entity_type   TEXT NOT NULL,       -- e.g. 'PROJECT', 'TASK'
    entity_id     UUID,
    prev_state    JSONB,
    new_state     JSONB,
    metadata      JSONB,
    ip_address    INET,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_project ON audit_logs(project_id);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);

-- Revoke destructive privileges on audit table (run as superuser at setup time)
-- REVOKE UPDATE, DELETE, TRUNCATE ON audit_logs FROM pms3_user;

-- ─── ROUTING TEMPLATES ────────────────────────────────────

CREATE TABLE routing_templates (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE routing_template_steps (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id       UUID NOT NULL REFERENCES routing_templates(id) ON DELETE CASCADE,
    step_order        INTEGER NOT NULL,
    dependency_policy dependency_policy NOT NULL DEFAULT 'REQUIRE_ALL',
    label             TEXT,
    department_ids    UUID[]    -- departments in this parallel step
);

-- ─── REFRESH TOKENS ───────────────────────────────────────

CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
