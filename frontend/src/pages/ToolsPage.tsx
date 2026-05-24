import { useState } from 'react'
import { Activity, CheckCircle2, Globe, Loader2, RefreshCw, Trash2, Wrench, XCircle } from 'lucide-react'
import { api } from '@/api/client'
import { mihomoApi } from '@/api/mihomo'
import { proxyApi } from '@/api/proxy'
import { cn } from '@/lib/utils'
import { useThemeStore } from '@/stores/themeStore'

type ConnectivityResult = {
  name: string
  url: string
  success: boolean
  statusCode?: number
  latencyMs: number
  error?: string
}

type DiagnosticItem = {
  name: string
  status: 'ok' | 'warn' | 'error'
  detail: string
}

type DiagnosticResult = {
  status: 'ok' | 'warn' | 'error'
  items: DiagnosticItem[]
}

export default function ToolsPage() {
  const { themeStyle } = useThemeStore()
  const [loading, setLoading] = useState<string | null>(null)
  const [connectivity, setConnectivity] = useState<ConnectivityResult[] | null>(null)
  const [diagnostics, setDiagnostics] = useState<DiagnosticResult | null>(null)
  const [chatgptCheck, setChatgptCheck] = useState<ConnectivityResult[] | null>(null)

  const handleReloadConfig = async () => {
    try {
      setLoading('reload')
      await mihomoApi.reloadConfig()
      alert('Reload success')
    } catch (e: unknown) {
      alert((e as Error)?.message || 'Reload failed')
    } finally {
      setLoading(null)
    }
  }

  const handleFlushDns = async () => {
    try {
      setLoading('dns')
      await mihomoApi.flushDns()
      alert('DNS cache flushed')
    } catch (e: unknown) {
      alert((e as Error)?.message || 'Flush failed')
    } finally {
      setLoading(null)
    }
  }

  const handleUpdateGeo = async () => {
    try {
      setLoading('geo')
      await mihomoApi.updateGeo()
      alert('Geo data updated')
    } catch (e: unknown) {
      alert((e as Error)?.message || 'Update failed')
    } finally {
      setLoading(null)
    }
  }

  const handleConnectivityTest = async () => {
    try {
      setLoading('connectivity')
      const result = await api.get<{ targets: ConnectivityResult[] }>('/system/connectivity')
      setConnectivity(result.targets)
    } catch (e: unknown) {
      alert((e as Error)?.message || 'Connectivity test failed')
    } finally {
      setLoading(null)
    }
  }

  const handleDiagnostics = async () => {
    try {
      setLoading('diagnostics')
      const result = await api.get<DiagnosticResult>('/system/diagnostics')
      setDiagnostics(result)
    } catch (e: unknown) {
      alert((e as Error)?.message || 'Diagnostics failed')
    } finally {
      setLoading(null)
    }
  }

  const handleRepairCore = async () => {
    try {
      setLoading('repair')
      await proxyApi.restart()
      await new Promise((resolve) => window.setTimeout(resolve, 2000))
      const result = await api.get<DiagnosticResult>('/system/diagnostics')
      setDiagnostics(result)
      alert('Core restart requested')
    } catch (e: unknown) {
      alert((e as Error)?.message || 'Repair failed')
    } finally {
      setLoading(null)
    }
  }

  const handleChatGPTCheck = async () => {
    try {
      setLoading('chatgpt')
      const result = await api.get<{ targets: ConnectivityResult[] }>('/system/chatgpt-check')
      setChatgptCheck(result.targets)
    } catch (e: unknown) {
      alert((e as Error)?.message || 'ChatGPT check failed')
    } finally {
      setLoading(null)
    }
  }

  const quickControls = [
    {
      id: 'reload',
      icon: RefreshCw,
      label: 'Reload core',
      description: 'Reload the current core config',
      color: 'orange',
      onClick: handleReloadConfig,
    },
    {
      id: 'dns',
      icon: Trash2,
      label: 'Flush DNS',
      description: 'Clear DNS cache in the proxy core',
      color: 'pink',
      onClick: handleFlushDns,
    },
    {
      id: 'geo',
      icon: Globe,
      label: 'Update Geo data',
      description: 'Update GeoIP and GeoSite data',
      color: 'blue',
      onClick: handleUpdateGeo,
    },
    {
      id: 'connectivity',
      icon: Globe,
      label: 'Test Google / ChatGPT',
      description: 'Check access through the local proxy',
      color: 'green',
      onClick: handleConnectivityTest,
    },
    {
      id: 'diagnostics',
      icon: Activity,
      label: 'Run diagnostics',
      description: 'Check service, core, ports, files, and DNS',
      color: 'violet',
      onClick: handleDiagnostics,
    },
    {
      id: 'repair',
      icon: Wrench,
      label: 'Repair proxy core',
      description: 'Restart the proxy core and run diagnostics',
      color: 'red',
      onClick: handleRepairCore,
    },
    {
      id: 'chatgpt',
      icon: Globe,
      label: 'ChatGPT check',
      description: 'Test ChatGPT and OpenAI domains via proxy',
      color: 'cyan',
      onClick: handleChatGPTCheck,
    },
  ]

  const getColorClasses = (color: string) => {
    const colors: Record<string, string> = {
      orange: themeStyle === 'apple-glass'
        ? 'bg-orange-500 text-white'
        : 'bg-orange-500/20 text-orange-400',
      pink: themeStyle === 'apple-glass'
        ? 'bg-pink-500 text-white'
        : 'bg-pink-500/20 text-pink-400',
      blue: themeStyle === 'apple-glass'
        ? 'bg-blue-500 text-white'
        : 'bg-blue-500/20 text-blue-400',
      green: themeStyle === 'apple-glass'
        ? 'bg-emerald-500 text-white'
        : 'bg-emerald-500/20 text-emerald-400',
      violet: themeStyle === 'apple-glass'
        ? 'bg-violet-500 text-white'
        : 'bg-violet-500/20 text-violet-400',
      red: themeStyle === 'apple-glass'
        ? 'bg-red-500 text-white'
        : 'bg-red-500/20 text-red-400',
      cyan: themeStyle === 'apple-glass'
        ? 'bg-cyan-500 text-white'
        : 'bg-cyan-500/20 text-cyan-400',
    }
    return colors[color] || colors.blue
  }

  const renderConnectivityResults = (items: ConnectivityResult[]) => (
    <div className={cn(
      'mt-4 divide-y rounded-xl border',
      themeStyle === 'apple-glass'
        ? 'divide-slate-200 border-slate-200 bg-white/50'
        : 'divide-white/10 border-white/10 bg-white/5'
    )}>
      {items.map((item) => (
        <div key={item.name} className="flex items-center justify-between gap-4 p-4">
          <div className="flex min-w-0 items-center gap-3">
            {item.success ? (
              <CheckCircle2 className="h-5 w-5 flex-none text-emerald-500" />
            ) : (
              <XCircle className="h-5 w-5 flex-none text-red-500" />
            )}
            <div className="min-w-0">
              <div className={cn(
                'font-medium',
                themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'
              )}>
                {item.name}
              </div>
              <div className={cn(
                'break-all text-xs',
                themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400'
              )}>
                {item.error || item.url}
              </div>
            </div>
          </div>
          <div className={cn(
            'flex-none text-right text-sm',
            themeStyle === 'apple-glass' ? 'text-slate-600' : 'text-slate-300'
          )}>
            <div>{item.success ? 'OK' : 'Failed'}</div>
            <div className="text-xs opacity-70">
              {item.statusCode ? `HTTP ${item.statusCode} / ` : ''}{item.latencyMs} ms
            </div>
          </div>
        </div>
      ))}
    </div>
  )

  return (
    <div className="space-y-6">
      <div className="glass-card p-6">
        <h2 className={cn(
          'mb-4 text-lg font-semibold',
          themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'
        )}>
          Quick controls
        </h2>

        <div className="space-y-3">
          {quickControls.map((control) => {
            const Icon = control.icon
            const isLoading = loading === control.id

            return (
              <button
                key={control.id}
                onClick={control.onClick}
                disabled={loading !== null}
                className={cn(
                  'flex w-full items-center gap-4 rounded-xl p-4 transition-all',
                  themeStyle === 'apple-glass'
                    ? 'border border-black/5 bg-white/60 hover:bg-white/80'
                    : 'border border-white/5 bg-white/5 hover:bg-white/10',
                  loading !== null && 'cursor-not-allowed opacity-50'
                )}
              >
                <div className={cn(
                  'flex h-12 w-12 items-center justify-center rounded-xl',
                  getColorClasses(control.color)
                )}>
                  {isLoading ? (
                    <Loader2 className="h-5 w-5 animate-spin" />
                  ) : (
                    <Icon className="h-5 w-5" />
                  )}
                </div>
                <div className="min-w-0 text-left">
                  <div className={cn(
                    'font-medium',
                    themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'
                  )}>
                    {control.label}
                  </div>
                  <div className={cn(
                    'text-sm',
                    themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400'
                  )}>
                    {control.description}
                  </div>
                </div>
              </button>
            )
          })}
        </div>

        {diagnostics && (
          <div className={cn(
            'mt-4 divide-y rounded-xl border',
            themeStyle === 'apple-glass'
              ? 'divide-slate-200 border-slate-200 bg-white/50'
              : 'divide-white/10 border-white/10 bg-white/5'
          )}>
            <div className="flex items-center justify-between gap-4 p-4">
              <div className={cn(
                'font-medium',
                themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'
              )}>
                Diagnostics
              </div>
              <div className={cn(
                'rounded-full px-2 py-1 text-xs font-medium uppercase',
                diagnostics.status === 'ok' && 'bg-emerald-500/20 text-emerald-500',
                diagnostics.status === 'warn' && 'bg-amber-500/20 text-amber-500',
                diagnostics.status === 'error' && 'bg-red-500/20 text-red-500'
              )}>
                {diagnostics.status}
              </div>
            </div>
            {diagnostics.items.map((item) => (
              <div key={item.name} className="flex items-center justify-between gap-4 p-4">
                <div className="flex min-w-0 items-center gap-3">
                  {item.status === 'ok' ? (
                    <CheckCircle2 className="h-5 w-5 flex-none text-emerald-500" />
                  ) : (
                    <XCircle className={cn(
                      'h-5 w-5 flex-none',
                      item.status === 'warn' ? 'text-amber-500' : 'text-red-500'
                    )} />
                  )}
                  <div className="min-w-0">
                    <div className={cn(
                      'font-medium',
                      themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'
                    )}>
                      {item.name}
                    </div>
                    <div className={cn(
                      'break-all text-xs',
                      themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400'
                    )}>
                      {item.detail}
                    </div>
                  </div>
                </div>
                <div className={cn(
                  'flex-none text-right text-sm',
                  themeStyle === 'apple-glass' ? 'text-slate-600' : 'text-slate-300'
                )}>
                  {item.status.toUpperCase()}
                </div>
              </div>
            ))}
          </div>
        )}

        {connectivity && renderConnectivityResults(connectivity)}
        {chatgptCheck && renderConnectivityResults(chatgptCheck)}
      </div>

      <div className="glass-card p-6">
        <h2 className={cn(
          'mb-4 text-lg font-semibold',
          themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'
        )}>
          More tools
        </h2>
        <p className={cn(
          'text-sm',
          themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400'
        )}>
          More features are coming soon.
        </p>
      </div>
    </div>
  )
}
