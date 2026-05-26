import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { Toaster } from 'sonner'
import { lazy, Suspense, useState, useEffect, ReactNode } from 'react'
import { Loader2 } from 'lucide-react'
import { Layout } from '@/components/layout'
import { authApi } from '@/api/auth'

const DashboardPage = lazy(() => import('@/pages/DashboardPage'))
const ProxySwitchPage = lazy(() => import('@/pages/ProxySwitchPage'))
const NodesPage = lazy(() => import('@/pages/NodesPage'))
const SubscriptionsPage = lazy(() => import('@/pages/SubscriptionsPage'))
const ConnectionsPage = lazy(() => import('@/pages/ConnectionsPage'))
const LogsPage = lazy(() => import('@/pages/LogsPage'))
const RulesetPage = lazy(() => import('@/pages/RulesetPage'))
const SettingsPage = lazy(() => import('@/pages/SettingsPage'))
const ToolsPage = lazy(() => import('@/pages/ToolsPage'))
const ConfigGeneratorPage = lazy(() => import('@/pages/ConfigGeneratorPage'))
const CoreManagePage = lazy(() => import('@/pages/CoreManagePage'))
const ProxySettingsPage = lazy(() => import('@/pages/ProxySettingsPage'))
const LoginPage = lazy(() => import('@/pages/LoginPage'))
const WireGuardPage = lazy(() => import('@/pages/WireGuardPage'))
const LegalPage = lazy(() => import('@/pages/LegalPage'))
const SingBoxSettingsPage = lazy(() => import('@/pages/SingBoxSettingsPage'))
const SingBoxRulesetPage = lazy(() => import('@/pages/SingBoxRulesetPage'))

function PageLoader() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Loader2 className="w-8 h-8 animate-spin text-primary" />
    </div>
  )
}

// Auth guard component
function AuthGuard({ children }: { children: ReactNode }) {
  const location = useLocation()
  const [checking, setChecking] = useState(true)
  const [needLogin, setNeedLogin] = useState(false)

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const status = await authApi.check()
        if (status.enabled && !status.authenticated) {
          setNeedLogin(true)
        }
      } catch {
        // Ignore errors
      } finally {
        setChecking(false)
      }
    }
    checkAuth()
  }, [location.pathname])

  if (checking) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  if (needLogin) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

function App() {
  return (
    <>
      <Suspense fallback={<PageLoader />}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/*" element={
            <AuthGuard>
              <Layout>
                <Suspense fallback={<PageLoader />}>
                  <Routes>
                    <Route path="/" element={<DashboardPage />} />
                    <Route path="/proxy-switch" element={<ProxySwitchPage />} />
                    <Route path="/nodes" element={<NodesPage />} />
                    <Route path="/subscriptions" element={<SubscriptionsPage />} />
                    <Route path="/connections" element={<ConnectionsPage />} />
                    <Route path="/logs" element={<LogsPage />} />
                    <Route path="/ruleset" element={<RulesetPage />} />
                    <Route path="/tools" element={<ToolsPage />} />
                    <Route path="/config-generator" element={<ConfigGeneratorPage />} />
                    <Route path="/core-manage" element={<CoreManagePage />} />
                    <Route path="/proxy-settings" element={<ProxySettingsPage />} />
                    <Route path="/singbox-settings" element={<SingBoxSettingsPage />} />
                    <Route path="/singbox-ruleset" element={<SingBoxRulesetPage />} />
                    <Route path="/settings" element={<SettingsPage />} />
                    <Route path="/wireguard" element={<WireGuardPage />} />
                    <Route path="/legal" element={<LegalPage />} />
                  </Routes>
                </Suspense>
              </Layout>
            </AuthGuard>
          } />
        </Routes>
      </Suspense>
      <Toaster position="bottom-right" richColors />
    </>
  )
}

export default App
