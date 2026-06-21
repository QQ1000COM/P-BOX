import { useState, useEffect, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, X, Trash2 } from 'lucide-react'
import { mihomoApi } from '@/api/mihomo'
import { formatBytes, cn } from '@/lib/utils'
import { useThemeStore } from '@/stores/themeStore'

interface Connection {
  id: string
  metadata: {
    host: string
    destinationPort: string
    sourceIP: string
    sourcePort: string
    network: string
    type: string
  }
  upload: number
  download: number
  start: string
  chains: string[]
  rule: string
  rulePayload: string
}

const MAX_STORED_CONNECTIONS = 500
const MAX_RENDERED_CONNECTIONS = 200
const CONNECTION_UPDATE_INTERVAL_MS = 1500

export default function ConnectionsPage() {
  const { t } = useTranslation()
  const { themeStyle } = useThemeStore()
  const [connections, setConnections] = useState<Connection[]>([])
  const [connectionCount, setConnectionCount] = useState(0)
  const [truncated, setTruncated] = useState(false)
  const [search, setSearch] = useState('')
  const [totalUp, setTotalUp] = useState(0)
  const [totalDown, setTotalDown] = useState(0)
  const lastUpdateRef = useRef(0)

  useEffect(() => {
    let ws: WebSocket | null = null
    let pollInterval: NodeJS.Timeout | null = null
    let usePolling = false

    const handleData = (data: { 
      connections?: Connection[]
      connectionCount?: number
      truncated?: boolean
      uploadTotal?: number
      downloadTotal?: number 
    }) => {
      const now = Date.now()
      if (now - lastUpdateRef.current < CONNECTION_UPDATE_INTERVAL_MS) {
        return
      }
      lastUpdateRef.current = now
      if (data.connections) {
        const topConnections = [...data.connections]
          .sort((a, b) => (b.upload + b.download) - (a.upload + a.download))
          .slice(0, MAX_STORED_CONNECTIONS)
        setConnections(topConnections)
        setConnectionCount(data.connectionCount ?? data.connections.length)
        setTruncated(Boolean(data.truncated || data.connections.length > topConnections.length))
      } else if (typeof data.connectionCount === 'number') {
        setConnectionCount(data.connectionCount)
      }
      if (data.uploadTotal !== undefined) {
        setTotalUp(data.uploadTotal)
      }
      if (data.downloadTotal !== undefined) {
        setTotalDown(data.downloadTotal)
      }
    }

    // HTTP 轮询作为后备
    const startPolling = () => {
      if (pollInterval) return
      usePolling = true
      const poll = async () => {
        try {
          const data = await mihomoApi.getConnections()
          handleData(data as { connections?: Connection[], connectionCount?: number, uploadTotal?: number, downloadTotal?: number })
        } catch {
          // ignore
        }
      }
      poll()
      pollInterval = setInterval(poll, 2500)
    }

    // 优先使用 WebSocket
    try {
      ws = mihomoApi.createConnectionsWs((data) => {
        handleData(data as { connections?: Connection[], connectionCount?: number, truncated?: boolean, uploadTotal?: number, downloadTotal?: number })
      }, { limit: MAX_STORED_CONNECTIONS, interval: CONNECTION_UPDATE_INTERVAL_MS })
      ws.onerror = () => {
        if (!usePolling) startPolling()
      }
      ws.onclose = () => {
        if (!usePolling) startPolling()
      }
    } catch {
      startPolling()
    }

    return () => {
      ws?.close()
      if (pollInterval) clearInterval(pollInterval)
    }
  }, [])

  const handleCloseAll = async () => {
    try {
      await mihomoApi.closeAllConnections()
    } catch {
      // Ignore errors
    }
  }

  const handleClose = async (id: string) => {
    try {
      await mihomoApi.closeConnection(id)
    } catch {
      // Ignore errors
    }
  }

  const filteredConnections = useMemo(() => connections.filter((conn) => {
    if (!search) return true
    const host = (conn.metadata?.host || '').toLowerCase()
    const rule = (conn.rule || '').toLowerCase()
    const searchLower = search.toLowerCase()
    return host.includes(searchLower) || rule.includes(searchLower)
  }), [connections, search])

  const visibleConnections = filteredConnections.slice(0, MAX_RENDERED_CONNECTIONS)

  return (
    <div className="space-y-4">
      {/* Stats bar */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
        <div className={cn(
          "flex flex-wrap items-center gap-3 sm:gap-6 text-xs sm:text-sm",
          themeStyle === 'apple-glass' ? 'text-slate-600' : 'text-slate-400'
        )}>
          <div>
            <span>{t('connections.active')}: </span>
            <span className={cn(
              "font-medium",
              themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'
            )}>{connectionCount || connections.length}</span>
          </div>
          <div>
            <span>{t('connections.upload')}: </span>
            <span className="font-medium text-blue-500">{formatBytes(totalUp)}</span>
          </div>
          <div>
            <span>{t('connections.download')}: </span>
            <span className="font-medium text-green-500">{formatBytes(totalDown)}</span>
          </div>
        </div>
        <button
          onClick={handleCloseAll}
          className="control-btn danger text-xs flex-shrink-0"
        >
          <Trash2 className="w-3 h-3" />
          <span className="hidden sm:inline">{t('common.closeAll')}</span>
          <span className="sm:hidden">{t('common.close')}</span>
        </button>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none z-10" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t('connections.searchPlaceholder')}
          className="form-input !pl-10"
        />
      </div>

      {(truncated || filteredConnections.length > visibleConnections.length) && (
        <div className={cn(
          "rounded-lg px-3 py-2 text-xs",
          themeStyle === 'apple-glass' ? 'bg-white/40 text-slate-600' : 'bg-white/5 text-slate-400'
        )}>
          当前连接较多，页面仅展示流量最高的前 {visibleConnections.length} 条，完整连接数仍按上方统计。
        </div>
      )}

      {/* Connections table */}
      <div className="glass-card overflow-auto max-h-[calc(100vh-260px)]">
        <table className="data-table">
          <thead>
            <tr className="bg-white/5">
              <th>{t('connections.host')}</th>
              <th>{t('connections.network')}</th>
              <th>{t('connections.type')}</th>
              <th>{t('connections.chains')}</th>
              <th>{t('connections.rule')}</th>
              <th>{t('connections.upload')}</th>
              <th>{t('connections.download')}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {visibleConnections.map((conn) => (
              <tr key={conn.id} className="group">
                <td className="text-foreground">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-emerald-400/70 flex-shrink-0" />
                    <span className="truncate max-w-[200px]">
                      {conn.metadata?.host || '-'}:{conn.metadata?.destinationPort || '-'}
                    </span>
                  </div>
                </td>
                <td className="text-muted-foreground uppercase text-[10px]">
                  {conn.metadata?.network || '-'}
                </td>
                <td className="text-muted-foreground text-[10px]">
                  {conn.metadata?.type || '-'}
                </td>
                <td className="text-muted-foreground text-[10px]">
                  {conn.chains.join(' → ')}
                </td>
                <td>
                  <span className="badge info">{conn.rule}</span>
                </td>
                <td className="text-blue-400 text-xs">
                  {formatBytes(conn.upload)}
                </td>
                <td className="text-green-400 text-xs">
                  {formatBytes(conn.download)}
                </td>
                <td>
                  <button
                    onClick={() => handleClose(conn.id)}
                    className="opacity-0 group-hover:opacity-100 p-1 hover:bg-red-500/20 rounded transition-all"
                  >
                    <X className="w-3.5 h-3.5 text-red-400" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {filteredConnections.length === 0 && (
          <div className={cn(
            "text-center py-12 text-sm",
            themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-neutral-500'
          )}>
            {t('common.loading')}
          </div>
        )}
      </div>
    </div>
  )
}
