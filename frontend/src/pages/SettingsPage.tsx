import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { Palette, Info, Check, Server, Power, Zap, Rocket, Gauge, ArrowUpDown, Loader2, Lock, Shield, ChevronRight } from 'lucide-react'
import { useThemeStore, ThemeStyle } from '@/stores/themeStore'
import { systemApi, SystemConfig } from '@/api/system'
import { authApi } from '@/api/auth'
import { cn } from '@/lib/utils'

export default function SettingsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { themeStyle, setThemeStyle } = useThemeStore()
  const [sysConfig, setSysConfig] = useState<SystemConfig | null>(null)
  const [sysLoading, setSysLoading] = useState(true)
  const [authEnabled, setAuthEnabled] = useState(false)
  const [authUsername, setAuthUsername] = useState('admin')
  const [username, setUsername] = useState('admin')
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [authLoading, setAuthLoading] = useState(false)

  useEffect(() => { fetchSysConfig(); fetchAuthConfig() }, [])

  const fetchAuthConfig = async () => {
    try {
      const cfg = await authApi.getConfig()
      setAuthEnabled(cfg.enabled)
      setAuthUsername(cfg.username || 'admin')
      setUsername(cfg.username || 'admin')
    } catch {}
  }
  const handleAuthToggle = async () => {
    try {
      await authApi.setEnabled(!authEnabled)
      setAuthEnabled(!authEnabled)
      toast.success(t('common.saved'))
    } catch (err) {
      toast.error((err as Error).message || t('common.error'))
    }
  }
  const fetchSysConfig = async () => { try { setSysLoading(true); setSysConfig(await systemApi.getConfig()) } catch {} finally { setSysLoading(false) } }

  const handleSysToggle = async (key: keyof SystemConfig, setter: (e: boolean) => Promise<unknown>) => {
    if (!sysConfig) return
    try { await setter(!sysConfig[key]); setSysConfig({ ...sysConfig, [key]: !sysConfig[key] }) } catch {}
  }

  const handleOptimizeAll = async () => { try { await systemApi.optimizeAll(); await fetchSysConfig() } catch {} }

  const handleUsernameSave = async () => {
    const nextUsername = username.trim()
    if (nextUsername.length < 2) {
      toast.error(t('auth.usernameTooShort'))
      return
    }
    setAuthLoading(true)
    try {
      await authApi.updateUsername(nextUsername)
      setAuthUsername(nextUsername)
      toast.success(t('common.saved'))
    } catch (err) {
      toast.error((err as Error).message || t('common.error'))
    } finally {
      setAuthLoading(false)
    }
  }

  const handlePasswordSave = async () => {
    if (newPassword.length < 6) {
      toast.error('密码长度至少 6 位')
      return
    }
    if (newPassword !== confirmPassword) {
      toast.error(t('auth.passwordMismatch'))
      return
    }
    setAuthLoading(true)
    try {
      await authApi.changePassword(oldPassword, newPassword)
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      localStorage.removeItem('p-box-token')
      toast.success('密码已修改，请重新登录')
    } catch (err) {
      toast.error((err as Error).message || t('auth.changeFailed'))
    } finally {
      setAuthLoading(false)
    }
  }

  const themeStyles: { id: ThemeStyle; label: string; description: string }[] = [
    { id: 'apple-glass', label: t('settings.appleGlass'), description: t('settings.glassDescription') },
    { id: 'apple-pro-dark', label: t('settings.appleProDark'), description: t('settings.proDarkDescription') },
  ]

  const Toggle = ({ value, onChange }: { value: boolean; onChange: () => void }) => (
    <button onClick={onChange} className={cn('w-12 h-6 rounded-full transition-colors relative', value ? (themeStyle === 'apple-glass' ? 'bg-blue-500' : 'bg-cyan-500') : 'bg-slate-600')}>
      <span className={cn('absolute top-1 w-4 h-4 rounded-full bg-white transition-all', value ? 'left-7' : 'left-1')} />
    </button>
  )

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      {/* 主题 */}
      <div className="glass-card p-5">
        <h3 className={cn('text-sm font-medium mb-4 flex items-center gap-2', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}><Palette className={cn('w-4 h-4', themeStyle === 'apple-glass' ? 'text-blue-500' : 'text-cyan-400')} />{t('settings.themeStyle')}</h3>
        <div className="grid grid-cols-2 gap-3">
          {themeStyles.map((theme) => (
            <button key={theme.id} onClick={() => setThemeStyle(theme.id)} className={cn('relative p-3 rounded-xl text-left transition-all border-2', themeStyle === theme.id ? (themeStyle === 'apple-glass' ? 'bg-blue-500/10 border-blue-500/50' : 'bg-cyan-500/10 border-cyan-500/50') : (themeStyle === 'apple-glass' ? 'bg-black/[0.02] border-transparent' : 'bg-white/5 border-transparent'))}>
              {themeStyle === theme.id && <div className="absolute top-2 right-2"><Check className={cn('w-4 h-4', themeStyle === 'apple-glass' ? 'text-blue-500' : 'text-cyan-400')} /></div>}
              <div className={cn('font-medium text-sm', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>{theme.label}</div>
              <div className={cn('text-xs mt-0.5', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-neutral-400')}>{theme.description}</div>
            </button>
          ))}
        </div>
      </div>

      {/* 安全 */}
      <div className="glass-card p-5">
        <h3 className={cn('text-sm font-medium mb-4 flex items-center gap-2', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}><Lock className={cn('w-4 h-4', themeStyle === 'apple-glass' ? 'text-blue-500' : 'text-cyan-400')} />{t('settings.security')}</h3>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div><span className={cn('text-sm', themeStyle === 'apple-glass' ? 'text-slate-600' : 'text-slate-300')}>{t('settings.enableAuth')}</span><p className={cn('text-xs', themeStyle === 'apple-glass' ? 'text-slate-400' : 'text-slate-500')}>{t('settings.enableAuthDesc')}</p></div>
            <Toggle value={authEnabled} onChange={handleAuthToggle} />
          </div>
          <div className={cn('rounded-xl p-3 space-y-3', themeStyle === 'apple-glass' ? 'bg-black/[0.03]' : 'bg-white/5')}>
            <div>
              <label className={cn('block text-xs mb-1', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')}>{t('auth.changeUsername')}</label>
              <div className="flex gap-2">
                <input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className={cn('flex-1 min-w-0 px-3 py-2 rounded-lg text-sm border outline-none', themeStyle === 'apple-glass' ? 'bg-white/60 border-white/60 text-slate-800' : 'bg-neutral-900 border-neutral-700 text-white')}
                  placeholder={authUsername}
                />
                <button
                  onClick={handleUsernameSave}
                  disabled={authLoading || username.trim() === authUsername}
                  className={cn('px-3 py-2 rounded-lg text-sm font-medium text-white disabled:opacity-50', themeStyle === 'apple-glass' ? 'bg-blue-500 hover:bg-blue-600' : 'bg-cyan-500 hover:bg-cyan-600')}
                >
                  {t('common.save')}
                </button>
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
              <input
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder={t('auth.oldPassword')}
                className={cn('px-3 py-2 rounded-lg text-sm border outline-none', themeStyle === 'apple-glass' ? 'bg-white/60 border-white/60 text-slate-800 placeholder:text-slate-400' : 'bg-neutral-900 border-neutral-700 text-white placeholder:text-slate-500')}
              />
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder={t('auth.newPassword')}
                className={cn('px-3 py-2 rounded-lg text-sm border outline-none', themeStyle === 'apple-glass' ? 'bg-white/60 border-white/60 text-slate-800 placeholder:text-slate-400' : 'bg-neutral-900 border-neutral-700 text-white placeholder:text-slate-500')}
              />
              <input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder={t('auth.confirmPassword')}
                className={cn('px-3 py-2 rounded-lg text-sm border outline-none', themeStyle === 'apple-glass' ? 'bg-white/60 border-white/60 text-slate-800 placeholder:text-slate-400' : 'bg-neutral-900 border-neutral-700 text-white placeholder:text-slate-500')}
              />
            </div>
            <button
              onClick={handlePasswordSave}
              disabled={authLoading || !oldPassword || !newPassword || !confirmPassword}
              className={cn('w-full px-3 py-2 rounded-lg text-sm font-medium text-white disabled:opacity-50', themeStyle === 'apple-glass' ? 'bg-indigo-500 hover:bg-indigo-600' : 'bg-blue-600 hover:bg-blue-700')}
            >
              {t('auth.changePassword')}
            </button>
          </div>
          {/* 安全与隐私政策 */}
          <button
            onClick={() => navigate('/legal')}
            className={cn(
              'w-full flex items-center justify-between p-3 rounded-xl transition-all',
              themeStyle === 'apple-glass' 
                ? 'bg-gradient-to-r from-amber-500/10 to-orange-500/10 hover:from-amber-500/20 hover:to-orange-500/20 border border-amber-500/20' 
                : 'bg-gradient-to-r from-amber-500/10 to-orange-500/10 hover:from-amber-500/20 hover:to-orange-500/20 border border-amber-500/20'
            )}
          >
            <div className="flex items-center gap-3">
              <div className={cn('w-8 h-8 rounded-lg flex items-center justify-center', themeStyle === 'apple-glass' ? 'bg-amber-500/20' : 'bg-amber-500/20')}>
                <Shield className="w-4 h-4 text-amber-500" />
              </div>
              <div className="text-left">
                <div className={cn('text-sm font-medium', themeStyle === 'apple-glass' ? 'text-slate-700' : 'text-white')}>
                  {t('legal.securityPolicy')}
                </div>
                <div className={cn('text-xs', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')}>
                  {t('legal.securityPolicyDesc')}
                </div>
              </div>
            </div>
            <ChevronRight className={cn('w-5 h-5', themeStyle === 'apple-glass' ? 'text-slate-400' : 'text-slate-500')} />
          </button>
        </div>
      </div>

      {/* 系统 */}
      <div className="glass-card p-5 lg:col-span-2">
        <h3 className={cn('text-sm font-medium mb-4 flex items-center gap-2', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}><Server className={cn('w-4 h-4', themeStyle === 'apple-glass' ? 'text-blue-500' : 'text-cyan-400')} />{t('settings.systemSettings')}</h3>
        {sysLoading ? (<div className="flex items-center justify-center py-6"><Loader2 className={cn('w-6 h-6 animate-spin', themeStyle === 'apple-glass' ? 'text-blue-500' : 'text-cyan-400')} /></div>) : sysConfig ? (
          <div className="space-y-4">
            <div className={cn('flex items-center justify-between p-4 rounded-xl', themeStyle === 'apple-glass' ? 'bg-gradient-to-r from-blue-500/10 to-purple-500/10 border border-blue-500/20' : 'bg-gradient-to-r from-cyan-500/10 to-purple-500/10 border border-cyan-500/20')}>
              <div className="flex items-center gap-3"><div className={cn('w-10 h-10 rounded-full flex items-center justify-center', themeStyle === 'apple-glass' ? 'bg-blue-500/20' : 'bg-cyan-500/20')}><Rocket className={cn('w-5 h-5', themeStyle === 'apple-glass' ? 'text-blue-500' : 'text-cyan-400')} /></div><div><div className={cn('font-medium', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>{t('settings.oneClickOptimize')}</div><div className={cn('text-xs', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')}>{t('settings.optimizeDesc')}</div></div></div>
              <button onClick={handleOptimizeAll} className={cn('flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-white', themeStyle === 'apple-glass' ? 'bg-blue-500 hover:bg-blue-600' : 'bg-cyan-500 hover:bg-cyan-600')}><Zap className="w-4 h-4" />{t('settings.optimizeNow')}</button>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {[{ key: 'autoStart' as const, icon: Power, label: t('settings.autoStart'), desc: t('settings.autoStartDesc'), setter: systemApi.setAutoStart },{ key: 'ipForward' as const, icon: ArrowUpDown, label: t('settings.ipForward'), desc: t('settings.ipForwardDesc'), setter: systemApi.setIPForward },{ key: 'bbrEnabled' as const, icon: Gauge, label: t('settings.bbr'), desc: t('settings.bbrDesc'), setter: systemApi.setBBR },{ key: 'tunOptimized' as const, icon: Rocket, label: t('settings.tunOptimize'), desc: t('settings.tunOptimizeDesc'), setter: systemApi.setTUNOptimize }].map(({ key, icon: Icon, label, desc, setter }) => (
                <div key={key} className="flex items-center justify-between"><div className="flex items-center gap-3"><Icon className={cn('w-4 h-4', themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-slate-400')} /><div><span className={cn('text-sm', themeStyle === 'apple-glass' ? 'text-slate-600' : 'text-slate-300')}>{label}</span><p className={cn('text-xs', themeStyle === 'apple-glass' ? 'text-slate-400' : 'text-slate-500')}>{desc}</p></div></div><Toggle value={sysConfig[key]} onChange={() => handleSysToggle(key, setter)} /></div>
              ))}
            </div>
            <p className={cn('text-xs text-center', themeStyle === 'apple-glass' ? 'text-slate-400' : 'text-slate-500')}>{t('settings.sysNote')}</p>
          </div>
        ) : (<div className={cn('text-center py-6', themeStyle === 'apple-glass' ? 'text-slate-400' : 'text-slate-500')}><Server className="w-10 h-10 mx-auto mb-3 opacity-50" /><p className="text-sm">{t('settings.sysNotAvailable')}</p></div>)}
      </div>

      {/* 关于 */}
      <div className="glass-card p-5 lg:col-span-2">
        <h3 className={cn('text-sm font-medium mb-4 flex items-center gap-2', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}><Info className={cn('w-4 h-4', themeStyle === 'apple-glass' ? 'text-blue-500' : 'text-cyan-400')} />{t('settings.about')}</h3>
        <div className="grid grid-cols-3 gap-4 text-sm">
          <div><span className={themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-neutral-500'}>{t('settings.version')}</span><p className={cn('font-mono', themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white')}>v1.0</p></div>
          <div><span className={themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-neutral-500'}>{t('settings.frontend')}</span><p className={themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'}>React + TS</p></div>
          <div><span className={themeStyle === 'apple-glass' ? 'text-slate-500' : 'text-neutral-500'}>{t('settings.uiLibrary')}</span><p className={themeStyle === 'apple-glass' ? 'text-slate-800' : 'text-white'}>Tailwind</p></div>
        </div>
      </div>
    </div>
  )
}
