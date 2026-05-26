import axios from 'axios'

const client = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

export interface CoreInfo {
  name: string
  version: string
  latestVersion: string
  installed: boolean
  path: string
}

export interface CoreStatus {
  currentCore: 'mihomo' | 'singbox'
  cores: Record<string, CoreInfo>
}

export interface DownloadProgress {
  downloading: boolean
  progress: number
  speed: number
  error?: string
}

export interface CoreHealth {
  coreType: string
  name: string
  path: string
  installed: boolean
  version: string
  latestVersion: string
  executable: boolean
  healthy: boolean
  issues: string[]
}

export interface UpdateSources {
  githubMirror: string
}

export const coreApi = {
  // Get core status
  getStatus: async (): Promise<CoreStatus> => {
    const res = await client.get('/core/status')
    return res.data.data
  },

  // Get latest versions
  getLatestVersions: async (): Promise<Record<string, string>> => {
    const res = await client.get('/core/versions')
    return res.data.data
  },

  // Switch core
  switchCore: async (coreType: string): Promise<void> => {
    await client.post('/core/switch', { coreType })
  },

  // Download core
  downloadCore: async (coreType: string): Promise<void> => {
    await client.post(`/core/download/${coreType}`)
  },

  // Get download progress
  getDownloadProgress: async (coreType: string): Promise<DownloadProgress> => {
    const res = await client.get(`/core/download/${coreType}/progress`)
    return res.data.data
  },

  getHealth: async (): Promise<Record<string, CoreHealth>> => {
    const res = await client.get('/core/health')
    return res.data.data
  },

  repairCore: async (coreType: string): Promise<void> => {
    await client.post(`/core/repair/${coreType}`)
  },

  getSources: async (): Promise<UpdateSources> => {
    const res = await client.get('/core/sources')
    return res.data.data
  },

  saveSources: async (sources: UpdateSources): Promise<void> => {
    await client.put('/core/sources', sources)
  },

  // Refresh versions (手动刷新版本信息)
  refreshVersions: async (): Promise<Record<string, string>> => {
    const res = await client.post('/core/versions/refresh')
    return res.data.data
  },

  // Get platform info
  getPlatformInfo: async (): Promise<{ os: string; arch: string }> => {
    const res = await client.get('/core/platform')
    return res.data.data
  },
}

export async function fetchLatestVersions(): Promise<{ mihomo: string; singbox: string }> {
  const versions = await coreApi.getLatestVersions()
  return {
    mihomo: versions.mihomo || '',
    singbox: versions.singbox || '',
  }
}
