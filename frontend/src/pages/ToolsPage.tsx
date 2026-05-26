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

  const copyText = async (text: string) => {
    await navigator.clipboard.writeText(text)
    alert('Copied')
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
      setTransformOutput((e as Error)?.message || 'Transform failed')
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
    { id: 'reload', icon: RefreshCw, label: 'Reload core', description: 'Reload the current core config', color: 'orange', onClick: handleReloadConfig },
    { id: 'dns', icon: Trash2, label: 'Flush DNS', description: 'Clear DNS cache in the proxy core', color: 'pink', onClick: handleFlushDns },
    { id: 'geo', icon: Globe, label: 'Update Geo data', description: 'Update GeoIP and GeoSite data', color: 'blue', onClick: handleUpdateGeo },
    { id: 'connectivity', icon: Globe, label: 'Test Google / ChatGPT', description: 'Check access through the local proxy', color: 'green', onClick: handleConnectivityTest },
    { id: 'diagnostics', icon: Activity, label: 'Run diagnostics', description: 'Check service, core, ports, files, and DNS', color: 'violet', onClick: handleDiagnostics },
    { id: 'repair', icon: Wrench, label: 'Repair proxy core', description: 'Restart the proxy core and run diagnostics', color: 'red', onClick: handleRepairCore },
    { id: 'chatgpt', icon: Globe, label: 'ChatGPT check', description: 'Test ChatGPT and OpenAI domains via proxy', color: 'cyan', onClick: handleChatGPTCheck },
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
        <h2 className={cn('mb-4 text-lg font-semibold', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>
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
              <div className={cn('font-medium', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>Diagnostics</div>
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
      </div>

      <div className="glass-card p-6">
        <h2 className={cn('mb-4 text-lg font-semibold', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>
          More tools
        </h2>

        <div className="grid gap-4 xl:grid-cols-2">
          <section className={panelClass}>
            <div className="mb-3 flex items-center gap-2">
              <Terminal className="h-4 w-4 text-cyan-500" />
              <h3 className={labelClass}>Proxy quick setup</h3>
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
              <h3 className={labelClass}>Encode / decode</h3>
            </div>
            <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
              {[
                ['base64-decode', 'Base64 decode'],
                ['base64-encode', 'Base64 encode'],
                ['url-decode', 'URL decode'],
                ['url-encode', 'URL encode'],
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
              placeholder="Paste text here"
            />
            <div className="mt-3 flex gap-2">
              <button className="control-btn text-xs" onClick={runTransform}>Run</button>
              <button className="control-btn text-xs" onClick={() => copyText(transformOutput)} disabled={!transformOutput}>
                <Clipboard className="h-3.5 w-3.5" />
                Copy
              </button>
            </div>
            <textarea
              className="form-input mt-3 min-h-[92px] resize-y"
              value={transformOutput}
              onChange={(e) => setTransformOutput(e.target.value)}
              placeholder="Output"
            />
          </section>

          <section className={panelClass}>
            <div className="mb-3 flex items-center gap-2">
              <Link2 className="h-4 w-4 text-emerald-500" />
              <h3 className={labelClass}>Subscription analyzer</h3>
            </div>
            <textarea
              className="form-input min-h-[130px] resize-y"
              value={subscriptionInput}
              onChange={(e) => setSubscriptionInput(e.target.value)}
              placeholder="Paste subscription content or Base64 subscription"
            />
            <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
              <div className={panelClass}>
                <div className={mutedClass}>Nodes</div>
                <div className="text-lg font-semibold">{subscriptionSummary.lines.length}</div>
              </div>
              {Object.entries(subscriptionSummary.protocols).slice(0, 3).map(([protocol, count]) => (
                <div key={protocol} className={panelClass}>
                  <div className={mutedClass}>{protocol}</div>
                  <div className="text-lg font-semibold">{count}</div>
                </div>
              ))}
            </div>
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
                Copy decoded
              </button>
              <button className="control-btn text-xs" onClick={() => downloadText('subscription.txt', subscriptionSummary.decoded)} disabled={!subscriptionSummary.decoded}>
                <Download className="h-3.5 w-3.5" />
                Download
              </button>
            </div>
          </section>

          <section className={panelClass}>
            <div className="mb-3 flex items-center gap-2">
              <Clock3 className="h-4 w-4 text-orange-500" />
              <h3 className={labelClass}>Timestamp converter</h3>
            </div>
            <input
              className="form-input"
              value={timestampInput}
              onChange={(e) => setTimestampInput(e.target.value)}
              placeholder="Unix seconds or milliseconds"
            />
            <div className="mt-3 space-y-2">
              {timestampResult ? (
                <>
                  <button className="w-full text-left" onClick={() => copyText(timestampResult.local)}>
                    <div className={mutedClass}>Local</div>
                    <div className="break-all text-sm">{timestampResult.local}</div>
                  </button>
                  <button className="w-full text-left" onClick={() => copyText(timestampResult.iso)}>
                    <div className={mutedClass}>ISO</div>
                    <div className="break-all text-sm">{timestampResult.iso}</div>
                  </button>
                  <div className="grid grid-cols-2 gap-2">
                    <button className="text-left" onClick={() => copyText(String(timestampResult.seconds))}>
                      <div className={mutedClass}>Seconds</div>
                      <div className="text-sm">{timestampResult.seconds}</div>
                    </button>
                    <button className="text-left" onClick={() => copyText(String(timestampResult.milliseconds))}>
                      <div className={mutedClass}>Milliseconds</div>
                      <div className="text-sm">{timestampResult.milliseconds}</div>
                    </button>
                  </div>
                </>
              ) : (
                <div className={mutedClass}>Enter a valid timestamp.</div>
              )}
            </div>
          </section>

          <section className={cn(panelClass, 'xl:col-span-2')}>
            <div className="mb-3 flex items-center gap-2">
              <FileJson className="h-4 w-4 text-blue-500" />
              <h3 className={labelClass}>Config formatter</h3>
            </div>
            <div className="grid gap-3 lg:grid-cols-2">
              <textarea
                className="form-input min-h-[180px] resize-y font-mono text-xs"
                value={configInput}
                onChange={(e) => setConfigInput(e.target.value)}
                placeholder="Paste JSON, Clash YAML, or sing-box config"
              />
              <textarea
                className="form-input min-h-[180px] resize-y font-mono text-xs"
                value={configOutput}
                onChange={(e) => setConfigOutput(e.target.value)}
                placeholder="Formatted output"
              />
            </div>
            <div className="mt-3 flex flex-wrap gap-2">
              <button className="control-btn text-xs" onClick={formatConfig}>Format</button>
              <button className="control-btn text-xs" onClick={() => copyText(configOutput)} disabled={!configOutput}>
                <Clipboard className="h-3.5 w-3.5" />
                Copy
              </button>
              <button className="control-btn text-xs" onClick={() => downloadText('p-box-config.txt', configOutput)} disabled={!configOutput}>
                <Download className="h-3.5 w-3.5" />
                Download
              </button>
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}
