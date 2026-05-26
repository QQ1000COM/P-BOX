import api from './client'

export interface SystemConfig {
  autoStart: boolean
  ipForward: boolean
  bbrEnabled: boolean
  tunOptimized: boolean
}

export interface SystemInfo {
  name: string
  version: string
  buildTime: string
}

export interface SystemResources {
  os: string
  platform: string
  kernel: string
  arch: string
  cpuModel: string
  cpuCores: number
  cpuUsage: number
  memoryTotal: number
  memoryUsed: number
  memoryPercent: number
  diskTotal: number
  diskUsed: number
  diskPercent: number
  uptime: number
}

export interface GeoIPInfo {
  ip: string
  country: string
  countryCode: string
  region: string
  city: string
  isp: string
  org: string
}

export interface SnapshotInfo {
  id: string
  label: string
  createdAt: string
  files: number
}

export interface SecurityAuditItem {
  name: string
  status: 'ok' | 'warn' | 'error'
  detail: string
}

export interface RuleTemplate {
  name: string
  target: string
  rules: string[]
}

export const systemApi = {
  getInfo: () => api.get<SystemInfo>('/system/info'),
  getResources: () => api.get<SystemResources>('/system/resources'),
  getConfig: () => api.get<SystemConfig>('/system/config'),
  setAutoStart: (enabled: boolean) => api.put('/system/autostart', { enabled }),
  setIPForward: (enabled: boolean) => api.put('/system/ipforward', { enabled }),
  setBBR: (enabled: boolean) => api.put('/system/bbr', { enabled }),
  setTUNOptimize: (enabled: boolean) => api.put('/system/tunoptimize', { enabled }),
  optimizeAll: () => api.post('/system/optimize-all', {}),
  enableSystemProxy: (host: string, port: number) => api.post('/system/proxy/enable', { host, port }),
  disableSystemProxy: () => api.post('/system/proxy/disable', {}),
  getSystemProxyStatus: () => api.get<{ enabled: boolean; host: string; port: number }>('/system/proxy/status'),
  getGeoIP: (lang: string = 'zh') => api.get<GeoIPInfo>(`/system/geoip?lang=${lang}`),
  createSnapshot: (label: string) => api.post<SnapshotInfo>('/system/history/snapshot', { label }),
  listHistory: () => api.get<SnapshotInfo[]>('/system/history'),
  restoreSnapshot: (id: string) => api.post(`/system/history/${id}/restore`, {}),
  securityAudit: () => api.get<SecurityAuditItem[]>('/system/security-audit'),
  ruleTemplates: () => api.get<RuleTemplate[]>('/system/rule-templates'),
}
