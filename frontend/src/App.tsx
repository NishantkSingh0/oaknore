import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import AppLayout from '@/components/layout/AppLayout'
import LoginPage from '@/pages/auth/LoginPage'
import DashboardPage from '@/pages/admin/DashboardPage'
import ProjectListPage from '@/pages/projects/ProjectListPage'
import ProjectDetailPage from '@/pages/projects/ProjectDetailPage'
import ProjectCreatePage from '@/pages/projects/ProjectCreatePage'
import ProjectTimelinePage from '@/pages/projects/ProjectTimelinePage'
import TaskBoardPage from '@/pages/tasks/TaskBoardPage'
import TaskDetailPage from '@/pages/tasks/TaskDetailPage'
import RoutingBuilderPage from '@/pages/tasks/RoutingBuilderPage'
import IssueListPage from '@/pages/issues/IssueListPage'
import IssueDetailPage from '@/pages/issues/IssueDetailPage'
import ReworkListPage from '@/pages/issues/ReworkListPage'
import QueryPanelPage from '@/pages/issues/QueryPanelPage'
import DepartmentListPage from '@/pages/departments/DepartmentListPage'
import EmployeeListPage from '@/pages/employees/EmployeeListPage'
import ReportListPage from '@/pages/reports/ReportListPage'
import NotificationPage from '@/pages/notifications/NotificationPage'
import SearchPage from '@/pages/search/SearchPage'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()
  if (isLoading) return <LoadingSpinner fullScreen />
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <PrivateRoute>
            <AppLayout />
          </PrivateRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />

        {/* Projects */}
        <Route path="projects" element={<ProjectListPage />} />
        <Route path="projects/new" element={<ProjectCreatePage />} />
        <Route path="projects/:projectId" element={<ProjectDetailPage />} />
        <Route path="projects/:projectId/timeline" element={<ProjectTimelinePage />} />
        <Route path="projects/:projectId/routing" element={<RoutingBuilderPage />} />
        <Route path="projects/:projectId/tasks" element={<TaskBoardPage />} />
        <Route path="projects/:projectId/issues" element={<IssueListPage />} />
        <Route path="projects/:projectId/reworks" element={<ReworkListPage />} />
        <Route path="projects/:projectId/queries" element={<QueryPanelPage />} />
        <Route path="projects/:projectId/reports" element={<ReportListPage />} />

        {/* Standalone task */}
        <Route path="tasks/:taskId" element={<TaskDetailPage />} />

        {/* Standalone issue */}
        <Route path="issues/:issueId" element={<IssueDetailPage />} />

        {/* Org management */}
        <Route path="departments" element={<DepartmentListPage />} />
        <Route path="employees" element={<EmployeeListPage />} />

        {/* Misc */}
        <Route path="notifications" element={<NotificationPage />} />
        <Route path="search" element={<SearchPage />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
