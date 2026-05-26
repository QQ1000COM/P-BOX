import { useMemo, useState } from 'react'
import {
  Activity,
  CheckCircle2,
  Clipboard,
  Clock3,
  Download,
  FileJson,
  Globe,
  KeyRound,
  Link2,
  Loader2,
  RefreshCw,
  Terminal,
  Trash2,
  Wrench,
  XCircle,
} from 'lucide-react'
import { api } from '@/api/client'
import { coreApi, CoreHealth } from '@/api/core'
import { mihomoApi } from '@/api/mihomo'
import { proxyApi } from '@/api/proxy'
import { systemApi, RuleTemplate, SecurityAuditItem, SnapshotInfo } from '@/api/system'
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

type ConnectionAnalysis = {
  total: number
  uploadTotal: number
  downloadTotal: number
  topHosts: Array<{ host: string; count: number }>
}

type TransformMode = 'base64-decode' | 'base64-encode' | 'url-decode' | 'url-encode'

const PROXY_HOST = '127.0.0.1'
const MIXED_PORT = 7890

function safeDecodeBase64(value: string) {
  const normalized = value.trim().replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized + '='.repeat((4 - (normalized.length % 4)) % 4)
  const binary = atob(padded)
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

function safeEncodeBase64(value: string) {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return btoa(binary)
}

function maybeDecodeSubscription(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return ''
  try {
    const decoded = safeDecodeBase64(trimmed)
    return decoded.includes('://') ? decoded : trimmed
  } catch {
    return trimmed
  }
}

function parseNodeName(line: string) {
  try {
    const url = new URL(line)
    return decodeURIComponent(url.hash.replace(/^#/, '') || url.searchParams.get('remarks') || url.hostname)
  } catch {
    const hashIndex = line.indexOf('#')
    return hashIndex >= 0 ? decodeURIComponent(line.slice(hashIndex + 1)) : line.slice(0, 80)
  }
}

function summarizeSubscription(raw: string) {
  const decoded = maybeDecodeSubscription(raw)
  const lines = decoded
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
  const protocols = lines.reduce<Record<string, number>>((acc, line) => {
    const match = line.match(/^([a-z0-9+.-]+):\/\//i)
    const key = (match?.[1] || 'unknown').toLowerCase()
    acc[key] = (acc[key] || 0) + 1
    return acc
  }, {})
  return {
    decoded,
    lines,
    protocols,
    names: lines.slice(0, 8).map(parseNodeName),
  }
}

function downloadText(filename: string, content: string) {
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

export default function ToolsPage() {
  const { themeStyle } = useThemeStore()
  const [loading, setLoading] = useState<string | null>(null)
  const [connectivity, setConnectivity] = useState<ConnectivityResult[] | null>(null)
  const [diagnostics, setDiagnostics] = useState<DiagnosticResult | null>(null)
  const [chatgptCheck, setChatgptCheck] = useState<ConnectivityResult[] | null>(null)
  const [transformMode, setTransformMode] = useState<TransformMode>('base64-decode')
  const [transformInput, setTransformInput] = useState('')
  const [transformOutput, setTransformOutput] = useState('')
  const [subscriptionInput, setSubscriptionInput] = useState('')
  const [timestampInput, setTimestampInput] = useState(() => String(Math.floor(Date.now() / 1000)))
  const [configInput, setConfigInput] = useState('')
  const [configOutput, setConfigOutput] = useState('')
  const [coreHealth, setCoreHealth] = useState<Record<string, CoreHealth> | null>(null)
  const [githubMirror, setGithubMirror] = useState('')
  const [snapshots, setSnapshots] = useState<SnapshotInfo[]>([])
  const [securityAudit, setSecurityAudit] = useState<SecurityAuditItem[]>([])
  const [ruleTemplates, setRuleTemplates] = useState<RuleTemplate[]>([])
  const [connectionAnalysis, setConnectionAnalysis] = useState<ConnectionAnalysis | null>(null)

  const proxyCommands = useMemo(() => {
    const httpProxy = `http://${PROXY_HOST}:${MIXED_PORT}`
    const socksProxy = `socks5://${PROXY_HOST}:${MIXED_PORT}`
    return [
      `export http_proxy=${httpProxy} https_proxy=${httpProxy} all_proxy=${socksProxy}`,
      `set HTTP_PROXY=${httpProxy} && set HTTPS_PROXY=${httpProxy}`,
      `curl -x ${httpProxy} https://api.ipify.org`,
      `git config --global http.proxy ${httpProxy}`,
    ]
  }, [])

  const subscriptionSummary = useMemo(
    () => summarizeSubscription(subscriptionInput),
    [subscriptionInput]
  )

  const subscriptionQuality = useMemo(() => {
    const total = subscriptionSummary.lines.length
    const supported = subscriptionSummary.lines.filter((line) => /^(ss|ssr|vmess|vless|trojan|hysteria|hysteria2|tuic|naive|socks|http):\/\//i.test(line)).length
    const named = subscriptionSummary.names.filter((name) => name && name.length < 80).length
    const duplicates = total - new Set(subscriptionSummary.lines).size
    const score = total === 0 ? 0 : Math.max(0, Math.min(100, Math.round((supported / total) * 65 + (named / total) * 25 - duplicates * 2 + 10)))
    return { score, supported, duplicates }
  }, [subscriptionSummary])

  const timestampResult = useMemo(() => {
    const value = timestampInput.trim()
    if (!value) return null
    const numeric = Number(value)
    if (!Number.isFinite(numeric)) return null
    const ms = value.length <= 10 ? numeric * 1000 : numeric
    const date = new Date(ms)
    if (Number.isNaN(date.getTime())) return null
    return {
      iso: date.toISOString(),
      local: date.toLocaleString(),
      seconds: Math.floor(ms / 1000),
      milliseconds: ms,
    }
  }, [timestampInput])

  const handleReloadConfig = async () => {
    try {
      setLoading('reload')
      await mihomoApi.reloadConfig()
      alert('重载成功')
    } catch (e: unknown) {
      alert((e as Error)?.message || '重载失败')
    } finally {
      setLoading(null)
    }
  }

  const handleFlushDns = async () => {
    try {
      setLoading('dns')
      await mihomoApi.flushDns()
      alert('DNS 缓存已清空')
    } catch (e: unknown) {
      alert((e as Error)?.message || '清空失败')
    } finally {
      setLoading(null)
    }
  }

  const handleUpdateGeo = async () => {
    try {
      setLoading('geo')
      await mihomoApi.updateGeo()
      alert('Geo 数据已更新')
    } catch (e: unknown) {
      alert((e as Error)?.message || '更新失败')
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
      alert((e as Error)?.message || '连通性测试失败')
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
      alert((e as Error)?.message || '诊断失败')
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
      alert('已请求重启核心')
    } catch (e: unknown) {
      alert((e as Error)?.message || '修复失败')
    } finally {
      setLoading(null)
    }
  }

  const handleExportDiagnostics = async () => {
    try {
      setLoading('export')
      const [diagnosticResult, connectivityResult] = await Promise.all([
        api.get<DiagnosticResult>('/system/diagnostics'),
        api.get<{ targets: ConnectivityResult[] }>('/system/connectivity'),
      ])

      setDiagnostics(diagnosticResult)
      setConnectivity(connectivityResult.targets)

      const report = [
        '# P-BOX 诊断报告',
        `生成时间: ${new Date().toLocaleString()}`,
        '',
        `总体状态: ${diagnosticResult.status.toUpperCase()}`,
        '',
        '## 系统诊断',
        ...diagnosticResult.items.map((item) => `- [${item.status.toUpperCase()}] ${item.name}: ${item.detail}`),
        '',
        '## 连通性测试',
        ...connectivityResult.targets.map((item) => {
          const status = item.success ? '正常' : '失败'
          const http = item.statusCode ? ` HTTP ${item.statusCode}` : ''
          const detail = item.error || item.url
          return `- [${status}] ${item.name}: ${item.latencyMs} ms${http} - ${detail}`
        }),
        '',
      ].join('\n')

      const stamp = new Date().toISOString().replace(/[:.]/g, '-')
      downloadText(`p-box-diagnostics-${stamp}.md`, report)
    } catch (e: unknown) {
      alert((e as Error)?.message || '导出诊断报告失败')
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
      alert((e as Error)?.message || 'ChatGPT 检测失败')
    } finally {
      setLoading(null)
    }
  }

  const handleCoreHealth = async () => {
    try {
      setLoading('core-health')
      const result = await coreApi.getHealth()
      setCoreHealth(result)
      const sources = await coreApi.getSources()
      setGithubMirror(sources.githubMirror || '')
    } catch (e: unknown) {
      alert((e as Error)?.message || '核心健康检查失败')
    } finally {
      setLoading(null)
    }
  }

  const handleRepairAllCores = async () => {
    try {
      setLoading('repair-all')
      await coreApi.repairCore('mihomo')
      await coreApi.repairCore('singbox')
      setCoreHealth(await coreApi.getHealth())
      alert('核心修复完成')
    } catch (e: unknown) {
      alert((e as Error)?.message || '核心修复失败')
    } finally {
      setLoading(null)
    }
  }

  const handleSaveSources = async () => {
    try {
      setLoading('sources')
      await coreApi.saveSources({ githubMirror })
      alert('更新源已保存')
    } catch (e: unknown) {
      alert((e as Error)?.message || '保存更新源失败')
    } finally {
      setLoading(null)
    }
  }

  const handleBackup = async () => {
    try {
      setLoading('backup')
      const token = localStorage.getItem('p-box-token')
      const response = await fetch('/api/system/backup', {
        headers: token ? { Authorization: `Bearer ${token}` } : undefined,
      })
      if (!response.ok) throw new Error('备份下载失败')
      const blob = await response.blob()
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `p-box-backup-${new Date().toISOString().replace(/[:.]/g, '-')}.zip`
      link.click()
      URL.revokeObjectURL(url)
    } catch (e: unknown) {
      alert((e as Error)?.message || '备份失败')
    } finally {
      setLoading(null)
    }
  }

  const handleCreateSnapshot = async () => {
    try {
      setLoading('snapshot')
      await systemApi.createSnapshot('工具箱快照')
      setSnapshots(await systemApi.listHistory())
      alert('快照已创建')
    } catch (e: unknown) {
      alert((e as Error)?.message || '创建快照失败')
    } finally {
      setLoading(null)
    }
  }

  const handleLoadHistory = async () => {
    try {
      setLoading('history')
      setSnapshots(await systemApi.listHistory())
    } catch (e: unknown) {
      alert((e as Error)?.message || '加载历史失败')
    } finally {
      setLoading(null)
    }
  }

  const handleSecurityAudit = async () => {
    try {
      setLoading('security')
      setSecurityAudit(await systemApi.securityAudit())
    } catch (e: unknown) {
      alert((e as Error)?.message || '安全检查失败')
    } finally {
      setLoading(null)
    }
  }

  const handleRuleTemplates = async () => {
    try {
      setLoading('templates')
      setRuleTemplates(await systemApi.ruleTemplates())
    } catch (e: unknown) {
      alert((e as Error)?.message || '加载规则模板失败')
    } finally {
      setLoading(null)
    }
  }

  const handleConnectionAnalysis = async () => {
    try {
      setLoading('connections-analysis')
      const data = await mihomoApi.getConnections()
      const hostCounts = new Map<string, number>()
      for (const item of data.connections as Array<{ metadata?: { host?: string; destinationIP?: string } }>) {
        const host = item.metadata?.host || item.metadata?.destinationIP || '未知目标'
        hostCounts.set(host, (hostCounts.get(host) || 0) + 1)
      }
      const topHosts = Array.from(hostCounts.entries())
        .map(([host, count]) => ({ host, count }))
        .sort((a, b) => b.count - a.count)
        .slice(0, 8)
      setConnectionAnalysis({
        total: data.connections.length,
        uploadTotal: data.uploadTotal,
        downloadTotal: data.downloadTotal,
        topHosts,
      })
    } catch (e: unknown) {
      alert((e as Error)?.message || '连接分析失败，请确认 Mihomo 正在运行')
    } finally {
      setLoading(null)
    }
  }

  const copyText = async (text: string) => {
    await navigator.clipboard.writeText(text)
    alert('已复制')
  }

  const runTransform = () => {
    try {
      const input = transformInput
      const output = {
        'base64-decode': () => safeDecodeBase64(input),
        'base64-encode': () => safeEncodeBase64(input),
        'url-decode': () => decodeURIComponent(input),
        'url-encode': () => encodeURIComponent(input),
      }[transformMode]()
      setTransformOutput(output)
    } catch (e: unknown) {
      setTransformOutput((e as Error)?.message || '转换失败')
    }
  }

  const formatConfig = () => {
    const input = configInput.trim()
    if (!input) {
      setConfigOutput('')
      return
    }
    try {
      setConfigOutput(JSON.stringify(JSON.parse(input), null, 2))
    } catch {
      const cleaned = input
        .split(/\r?\n/)
        .map((line) => line.trimEnd())
        .join('\n')
      setConfigOutput(cleaned)
    }
  }

  const quickControls = [
    { id: 'reload', icon: RefreshCw, label: '重载核心', description: '重新加载当前核心配置', color: 'orange', onClick: handleReloadConfig },
    { id: 'dns', icon: Trash2, label: '清空 DNS', description: '清空代理核心的 DNS 缓存', color: 'pink', onClick: handleFlushDns },
    { id: 'geo', icon: Globe, label: '更新 Geo 数据', description: '更新 GeoIP 和 GeoSite 数据', color: 'blue', onClick: handleUpdateGeo },
    { id: 'connectivity', icon: Globe, label: '测试 Google / ChatGPT', description: '通过本地代理检查访问状态', color: 'green', onClick: handleConnectivityTest },
    { id: 'diagnostics', icon: Activity, label: '运行诊断', description: '检查服务、核心、端口、文件和 DNS', color: 'violet', onClick: handleDiagnostics },
    { id: 'export', icon: Download, label: '导出报告', description: '下载诊断和连通性测试结果', color: 'blue', onClick: handleExportDiagnostics },
    { id: 'repair', icon: Wrench, label: '修复代理核心', description: '重启代理核心并重新运行诊断', color: 'red', onClick: handleRepairCore },
    { id: 'chatgpt', icon: Globe, label: 'ChatGPT 检测', description: '通过代理测试 ChatGPT 和 OpenAI 域名', color: 'cyan', onClick: handleChatGPTCheck },
    { id: 'core-health', icon: Activity, label: '核心健康检查', description: '检查 Mihomo 和 Sing-box 版本、路径和权限', color: 'violet', onClick: handleCoreHealth },
    { id: 'repair-all', icon: Wrench, label: '一键修复核心', description: '重新下载并修复两个内置核心', color: 'red', onClick: handleRepairAllCores },
    { id: 'backup', icon: Download, label: '导出备份', description: '导出配置、订阅、规则和状态文件', color: 'green', onClick: handleBackup },
    { id: 'snapshot', icon: Clock3, label: '创建配置快照', description: '保存当前配置历史，便于回滚', color: 'orange', onClick: handleCreateSnapshot },
    { id: 'history', icon: FileJson, label: '查看配置历史', description: '加载已保存的配置快照列表', color: 'blue', onClick: handleLoadHistory },
    { id: 'security', icon: KeyRound, label: '安全模式检查', description: '检查认证、核心目录和更新源安全项', color: 'pink', onClick: handleSecurityAudit },
    { id: 'templates', icon: FileJson, label: '规则模板', description: '查看内置 AI、流媒体、广告拦截等模板', color: 'cyan', onClick: handleRuleTemplates },
    { id: 'connections-analysis', icon: Activity, label: '连接分析', description: '统计当前连接数量和热门目标域名', color: 'green', onClick: handleConnectionAnalysis },
  ]

  const getColorClasses = (color: string) => {
    const colors: Record<string, string> = {
      orange: themeStyle === 'apple-glass' ? 'bg-orange-500 text-white' : 'bg-orange-500/20 text-orange-400',
      pink: themeStyle === 'apple-glass' ? 'bg-pink-500 text-white' : 'bg-pink-500/20 text-pink-400',
      blue: themeStyle === 'apple-glass' ? 'bg-blue-500 text-white' : 'bg-blue-500/20 text-blue-400',
      green: themeStyle === 'apple-glass' ? 'bg-emerald-500 text-white' : 'bg-emerald-500/20 text-emerald-400',
      violet: themeStyle === 'apple-glass' ? 'bg-violet-500 text-white' : 'bg-violet-500/20 text-violet-400',
      red: themeStyle === 'apple-glass' ? 'bg-red-500 text-white' : 'bg-red-500/20 text-red-400',
      cyan: themeStyle === 'apple-glass' ? 'bg-cyan-500 text-white' : 'bg-cyan-500/20 text-cyan-400',
    }
    return colors[color] || colors.blue
  }

  const panelClass = cn(
    'rounded-xl border p-4',
    themeStyle === 'apple-glass'
      ? 'border-slate-200 bg-white/50'
      : 'border-white/10 bg-white/5'
  )

  const labelClass = cn(
    'text-sm font-medium',
    themeStyle === 'apple-glass' ? 'text-slate-700' : 'text-slate-200'
  )

  const mutedClass = cn(
    'text-xs',
    themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400'
  )

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
              <div className={cn('font-medium', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>
                {item.name}
              </div>
              <div className={cn('break-all text-xs', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')}>
                {item.error || item.url}
              </div>
            </div>
          </div>
          <div className={cn('flex-none text-right text-sm', themeStyle === 'apple-glass' ? 'text-slate-600' : 'text-slate-300')}>
            <div>{item.success ? '正常' : '失败'}</div>
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
        <h2 className={cn('mb-4 text-lg font-semibold', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>
          快速控制
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
                <div className={cn('flex h-12 w-12 items-center justify-center rounded-xl', getColorClasses(control.color))}>
                  {isLoading ? <Loader2 className="h-5 w-5 animate-spin" /> : <Icon className="h-5 w-5" />}
                </div>
                <div className="min-w-0 text-left">
                  <div className={cn('font-medium', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>
                    {control.label}
                  </div>
                  <div className={cn('text-sm', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')}>
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
              <div className={cn('font-medium', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>诊断结果</div>
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
                    <XCircle className={cn('h-5 w-5 flex-none', item.status === 'warn' ? 'text-amber-500' : 'text-red-500')} />
                  )}
                  <div className="min-w-0">
                    <div className={cn('font-medium', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>
                      {item.name}
                    </div>
                    <div className={cn('break-all text-xs', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')}>
                      {item.detail}
                    </div>
                  </div>
                </div>
                <div className={cn('flex-none text-right text-sm', themeStyle === 'apple-glass' ? 'text-slate-600' : 'text-slate-300')}>
                  {item.status.toUpperCase()}
                </div>
              </div>
            ))}
          </div>
        )}

        {connectivity && renderConnectivityResults(connectivity)}
        {chatgptCheck && renderConnectivityResults(chatgptCheck)}

        {(coreHealth || snapshots.length > 0 || securityAudit.length > 0 || ruleTemplates.length > 0 || connectionAnalysis) && (
          <div className="mt-4 grid gap-4 xl:grid-cols-2">
            {coreHealth && (
              <section className={panelClass}>
                <div className="mb-3 flex items-center justify-between gap-3">
                  <h3 className={labelClass}>核心健康</h3>
                  <button className="control-btn text-xs" onClick={handleSaveSources} disabled={loading !== null}>
                    保存更新源
                  </button>
                </div>
                <input
                  className="form-input mb-3"
                  value={githubMirror}
                  onChange={(e) => setGithubMirror(e.target.value)}
                  placeholder="GitHub 镜像，例如 https://ghfast.top/"
                />
                <div className="space-y-2">
                  {Object.values(coreHealth).map((core) => (
                    <div key={core.coreType} className={cn('rounded-lg border p-3', themeStyle === 'apple-glass' ? 'border-slate-200 bg-white/50' : 'border-white/10 bg-white/5')}>
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <div className={labelClass}>{core.name}</div>
                          <div className={cn('break-all text-xs', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')}>{core.path}</div>
                        </div>
                        <span className={cn('rounded-full px-2 py-1 text-xs', core.healthy ? 'bg-emerald-500/20 text-emerald-500' : 'bg-amber-500/20 text-amber-500')}>
                          {core.healthy ? '正常' : '需处理'}
                        </span>
                      </div>
                      <div className="mt-2 text-xs">
                        已安装: {core.installed ? `v${core.version}` : '否'} / 最新: {core.latestVersion || '未知'}
                      </div>
                      {core.issues.length > 0 && (
                        <div className="mt-2 space-y-1 text-xs text-amber-500">
                          {core.issues.map((issue) => <div key={issue}>- {issue}</div>)}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </section>
            )}

            {snapshots.length > 0 && (
              <section className={panelClass}>
                <h3 className={cn(labelClass, 'mb-3')}>配置历史</h3>
                <div className="space-y-2">
                  {snapshots.slice(0, 6).map((snapshot) => (
                    <div key={snapshot.id} className="flex items-center justify-between gap-3 text-sm">
                      <div>
                        <div>{snapshot.label}</div>
                        <div className={mutedClass}>{new Date(snapshot.createdAt).toLocaleString()} · {snapshot.files} 个文件</div>
                      </div>
                      <button
                        className="control-btn text-xs"
                        onClick={async () => {
                          await systemApi.restoreSnapshot(snapshot.id)
                          alert('快照已恢复')
                        }}
                      >
                        恢复
                      </button>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {securityAudit.length > 0 && (
              <section className={panelClass}>
                <h3 className={cn(labelClass, 'mb-3')}>安全检查</h3>
                <div className="space-y-2">
                  {securityAudit.map((item) => (
                    <div key={item.name} className="flex items-start gap-2 text-sm">
                      {item.status === 'ok' ? <CheckCircle2 className="mt-0.5 h-4 w-4 text-emerald-500" /> : <XCircle className="mt-0.5 h-4 w-4 text-amber-500" />}
                      <div>
                        <div>{item.name}</div>
                        <div className={mutedClass}>{item.detail}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {ruleTemplates.length > 0 && (
              <section className={panelClass}>
                <h3 className={cn(labelClass, 'mb-3')}>规则模板</h3>
                <div className="space-y-3">
                  {ruleTemplates.map((template) => (
                    <div key={template.name}>
                      <div className="flex items-center justify-between gap-3">
                        <div className={labelClass}>{template.name}</div>
                        <button className="control-btn text-xs" onClick={() => copyText(template.rules.join('\n'))}>复制</button>
                      </div>
                      <div className={cn('mt-1 rounded-lg p-2 font-mono text-xs', themeStyle === 'apple-glass' ? 'bg-slate-100 text-slate-600' : 'bg-black/30 text-slate-300')}>
                        {template.rules.slice(0, 3).join(' / ')}
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {connectionAnalysis && (
              <section className={panelClass}>
                <h3 className={cn(labelClass, 'mb-3')}>连接分析</h3>
                <div className="mb-3 grid grid-cols-3 gap-2 text-sm">
                  <div>
                    <div className={mutedClass}>连接数</div>
                    <div className="font-mono text-lg">{connectionAnalysis.total}</div>
                  </div>
                  <div>
                    <div className={mutedClass}>上传</div>
                    <div className="font-mono text-lg">{Math.round(connectionAnalysis.uploadTotal / 1024 / 1024)} MB</div>
                  </div>
                  <div>
                    <div className={mutedClass}>下载</div>
                    <div className="font-mono text-lg">{Math.round(connectionAnalysis.downloadTotal / 1024 / 1024)} MB</div>
                  </div>
                </div>
                <div className="space-y-2">
                  {connectionAnalysis.topHosts.map((item) => (
                    <div key={item.host} className="flex items-center justify-between gap-3 text-sm">
                      <span className="truncate">{item.host}</span>
                      <span className={mutedClass}>{item.count}</span>
                    </div>
                  ))}
                </div>
              </section>
            )}
          </div>
        )}
      </div>

      <div className="glass-card p-6">
        <h2 className={cn('mb-4 text-lg font-semibold', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>
          更多工具
        </h2>

        <div className="grid gap-4 xl:grid-cols-2">
          <section className={panelClass}>
            <div className="mb-3 flex items-center gap-2">
              <Terminal className="h-4 w-4 text-cyan-500" />
              <h3 className={labelClass}>代理快速设置</h3>
            </div>
            <div className="space-y-2">
              {proxyCommands.map((command) => (
                <div key={command} className="flex items-center gap-2">
                  <code className={cn(
                    'min-w-0 flex-1 overflow-x-auto rounded-lg px-3 py-2 text-xs',
                    themeStyle === 'apple-glass' ? 'bg-slate-100 text-slate-700' : 'bg-black/30 text-slate-200'
                  )}>
                    {command}
                  </code>
                  <button className="control-btn text-xs" onClick={() => copyText(command)}>
                    <Clipboard className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          </section>

          <section className={panelClass}>
            <div className="mb-3 flex items-center gap-2">
              <KeyRound className="h-4 w-4 text-violet-500" />
              <h3 className={labelClass}>编码 / 解码</h3>
            </div>
            <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
              {[
                ['base64-decode', 'Base64 解码'],
                ['base64-encode', 'Base64 编码'],
                ['url-decode', 'URL 解码'],
                ['url-encode', 'URL 编码'],
              ].map(([value, label]) => (
                <button
                  key={value}
                  onClick={() => setTransformMode(value as TransformMode)}
                  className={cn(
                    'rounded-lg px-3 py-2 text-xs transition',
                    transformMode === value
                      ? 'bg-blue-500 text-white'
                      : themeStyle === 'apple-glass'
                        ? 'bg-white/70 text-slate-600 hover:bg-white'
                        : 'bg-white/5 text-slate-300 hover:bg-white/10'
                  )}
                >
                  {label}
                </button>
              ))}
            </div>
            <textarea
              className="form-input min-h-[92px] resize-y"
              value={transformInput}
              onChange={(e) => setTransformInput(e.target.value)}
              placeholder="在这里粘贴文本"
            />
            <div className="mt-3 flex gap-2">
              <button className="control-btn text-xs" onClick={runTransform}>执行</button>
              <button className="control-btn text-xs" onClick={() => copyText(transformOutput)} disabled={!transformOutput}>
                <Clipboard className="h-3.5 w-3.5" />
                复制
              </button>
            </div>
            <textarea
              className="form-input mt-3 min-h-[92px] resize-y"
              value={transformOutput}
              onChange={(e) => setTransformOutput(e.target.value)}
              placeholder="输出结果"
            />
          </section>

          <section className={panelClass}>
            <div className="mb-3 flex items-center gap-2">
              <Link2 className="h-4 w-4 text-emerald-500" />
              <h3 className={labelClass}>订阅分析器</h3>
            </div>
            <textarea
              className="form-input min-h-[130px] resize-y"
              value={subscriptionInput}
              onChange={(e) => setSubscriptionInput(e.target.value)}
              placeholder="粘贴订阅内容或 Base64 订阅"
            />
            <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
              <div className={panelClass}>
                <div className={mutedClass}>节点数</div>
                <div className="text-lg font-semibold">{subscriptionSummary.lines.length}</div>
              </div>
              {Object.entries(subscriptionSummary.protocols).slice(0, 3).map(([protocol, count]) => (
                <div key={protocol} className={panelClass}>
                  <div className={mutedClass}>{protocol}</div>
                  <div className="text-lg font-semibold">{count}</div>
                </div>
              ))}
              <div className={panelClass}>
                <div className={mutedClass}>质量分</div>
                <div className="text-lg font-semibold">{subscriptionQuality.score}</div>
              </div>
            </div>
            {subscriptionSummary.lines.length > 0 && (
              <div className={cn('mt-3 text-xs', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')}>
                支持协议 {subscriptionQuality.supported}/{subscriptionSummary.lines.length}
                {subscriptionQuality.duplicates > 0 ? ` · 重复 ${subscriptionQuality.duplicates}` : ''}
              </div>
            )}
            {subscriptionSummary.names.length > 0 && (
              <div className="mt-3 space-y-1">
                {subscriptionSummary.names.map((name, index) => (
                  <div key={`${name}-${index}`} className={cn('truncate text-xs', themeStyle === 'apple-glass' ? 'text-slate-600' : 'text-slate-300')}>
                    {index + 1}. {name}
                  </div>
                ))}
              </div>
            )}
            <div className="mt-3 flex gap-2">
              <button className="control-btn text-xs" onClick={() => copyText(subscriptionSummary.decoded)} disabled={!subscriptionSummary.decoded}>
                <Clipboard className="h-3.5 w-3.5" />
                复制解码内容
              </button>
              <button className="control-btn text-xs" onClick={() => downloadText('subscription.txt', subscriptionSummary.decoded)} disabled={!subscriptionSummary.decoded}>
                <Download className="h-3.5 w-3.5" />
                下载
              </button>
            </div>
          </section>

          <section className={panelClass}>
            <div className="mb-3 flex items-center gap-2">
              <Clock3 className="h-4 w-4 text-orange-500" />
              <h3 className={labelClass}>时间戳转换器</h3>
            </div>
            <input
              className="form-input"
              value={timestampInput}
              onChange={(e) => setTimestampInput(e.target.value)}
              placeholder="Unix 秒级或毫秒级时间戳"
            />
            <div className="mt-3 space-y-2">
              {timestampResult ? (
                <>
                  <button className="w-full text-left" onClick={() => copyText(timestampResult.local)}>
                    <div className={mutedClass}>本地时间</div>
                    <div className="break-all text-sm">{timestampResult.local}</div>
                  </button>
                  <button className="w-full text-left" onClick={() => copyText(timestampResult.iso)}>
                    <div className={mutedClass}>ISO</div>
                    <div className="break-all text-sm">{timestampResult.iso}</div>
                  </button>
                  <div className="grid grid-cols-2 gap-2">
                    <button className="text-left" onClick={() => copyText(String(timestampResult.seconds))}>
                      <div className={mutedClass}>秒</div>
                      <div className="text-sm">{timestampResult.seconds}</div>
                    </button>
                    <button className="text-left" onClick={() => copyText(String(timestampResult.milliseconds))}>
                      <div className={mutedClass}>毫秒</div>
                      <div className="text-sm">{timestampResult.milliseconds}</div>
                    </button>
                  </div>
                </>
              ) : (
                <div className={mutedClass}>请输入有效的时间戳。</div>
              )}
            </div>
          </section>

          <section className={cn(panelClass, 'xl:col-span-2')}>
            <div className="mb-3 flex items-center gap-2">
              <FileJson className="h-4 w-4 text-blue-500" />
              <h3 className={labelClass}>配置格式化</h3>
            </div>
            <div className="grid gap-3 lg:grid-cols-2">
              <textarea
                className="form-input min-h-[180px] resize-y font-mono text-xs"
                value={configInput}
                onChange={(e) => setConfigInput(e.target.value)}
                placeholder="粘贴 JSON、Clash YAML 或 sing-box 配置"
              />
              <textarea
                className="form-input min-h-[180px] resize-y font-mono text-xs"
                value={configOutput}
                onChange={(e) => setConfigOutput(e.target.value)}
                placeholder="格式化后的输出"
              />
            </div>
            <div className="mt-3 flex flex-wrap gap-2">
              <button className="control-btn text-xs" onClick={formatConfig}>格式化</button>
              <button className="control-btn text-xs" onClick={() => copyText(configOutput)} disabled={!configOutput}>
                <Clipboard className="h-3.5 w-3.5" />
                复制
              </button>
              <button className="control-btn text-xs" onClick={() => downloadText('p-box-config.txt', configOutput)} disabled={!configOutput}>
                <Download className="h-3.5 w-3.5" />
                下载
              </button>
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}
