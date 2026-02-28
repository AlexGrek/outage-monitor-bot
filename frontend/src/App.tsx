import { useState, useEffect, useCallback } from 'react'
import { api } from './lib/api'
import { useTheme } from './hooks/useTheme'
import { ThemeToggle } from './components/dashboard/ThemeToggle'
import { HealthBadge } from './components/dashboard/HealthBadge'
import { StatusCard } from './components/dashboard/StatusCard'
import { ConfigPanel } from './components/dashboard/ConfigPanel'
import { AutoRestartInfo } from './components/dashboard/AutoRestartInfo'
import { ApiKeyModal } from './components/dashboard/ApiKeyModal'
import { SourcesPanel } from './components/dashboard/SourcesPanel'
import { TabNavigation, type TabId } from './components/dashboard/TabNavigation'
import { SinksPanel } from './components/dashboard/SinksPanel'
import { EventsPanel } from './components/dashboard/EventsPanel'
import { ToastContainer, type ToastMessage } from './components/dashboard/Toast'
import type {
  HealthResponse,
  StatusResponse,
  ConfigResponse,
  Source,
  CreateSourceRequest,
  UpdateSourceRequest,
} from './types'

function App() {
  useTheme()
  const [activeTab, setActiveTab] = useState<TabId>('status')
  const [showApiKeyModal, setShowApiKeyModal] = useState(false)
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [status, setStatus] = useState<StatusResponse | null>(null)
  const [config, setConfig] = useState<ConfigResponse | null>(null)
  const [sources, setSources] = useState<Source[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [reloading, setReloading] = useState(false)
  const [toasts, setToasts] = useState<ToastMessage[]>([])

  const addToast = useCallback((toast: Omit<ToastMessage, 'id'>) => {
    const id = `${Date.now()}-${Math.random()}`
    setToasts((prev) => [...prev, { ...toast, id }])
  }, [])

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const loadData = useCallback(async () => {
    try {
      setError(null)
      // Always load health (no auth required)
      const healthData = await api.getHealth()
      setHealth(healthData)

      // Try to load authenticated data
      if (api.getApiKey()) {
        try {
          const [statusData, configData, sourcesData] = await Promise.all([
            api.getStatus(),
            api.getAllConfig(),
            api.getSources(),
          ])
          setStatus(statusData)
          setConfig(configData)
          setSources(sourcesData)
        } catch (err) {
          // If auth fails, show API key modal
          if (err instanceof Error && err.message.includes('401')) {
            setShowApiKeyModal(true)
          } else {
            throw err
          }
        }
      } else {
        setShowApiKeyModal(true)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadData()
    // Refresh data every 5 seconds
    const interval = setInterval(loadData, 5000)
    return () => clearInterval(interval)
  }, [loadData])

  const handleApiKeySubmit = (apiKey: string) => {
    api.setApiKey(apiKey)
    setShowApiKeyModal(false)
    setLoading(true)
    loadData()
  }

  const handleConfigUpdate = async (key: string, value: string) => {
    await api.updateConfig(key, value)
    // Reload data after a short delay to show new config
    setTimeout(loadData, 1000)
  }

  const handleReload = async () => {
    setReloading(true)
    try {
      await api.reloadBot()
      // Reload data after a short delay
      setTimeout(loadData, 2000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reload bot')
    } finally {
      setReloading(false)
    }
  }

  const handleClearApiKey = () => {
    api.clearApiKey()
    setShowApiKeyModal(true)
    setStatus(null)
    setConfig(null)
    setSources(null)
  }

  const handleCreateSource = async (data: CreateSourceRequest) => {
    await api.createSource(data)
    // Reload sources immediately
    const sourcesData = await api.getSources()
    setSources(sourcesData)
  }

  const handleUpdateSource = async (id: string, data: UpdateSourceRequest) => {
    await api.updateSource(id, data)
    // Reload sources immediately
    const sourcesData = await api.getSources()
    setSources(sourcesData)
  }

  const handleDeleteSource = async (id: string) => {
    await api.deleteSource(id)
    // Reload sources immediately
    const sourcesData = await api.getSources()
    setSources(sourcesData)
  }

  const handlePauseSource = async (id: string) => {
    await api.pauseSource(id)
    // Reload sources immediately
    const sourcesData = await api.getSources()
    setSources(sourcesData)
  }

  const handleResumeSource = async (id: string) => {
    await api.resumeSource(id)
    // Reload sources immediately
    const sourcesData = await api.getSources()
    setSources(sourcesData)
  }

  const formatUptime = (seconds: number): string => {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const mins = Math.floor((seconds % 3600) / 60)

    if (days > 0) return `${days}d ${hours}h`
    if (hours > 0) return `${hours}h ${mins}m`
    return `${mins}m`
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <ApiKeyModal isOpen={showApiKeyModal} onClose={handleApiKeySubmit} />
      <ToastContainer toasts={toasts} onClose={removeToast} />

      {/* Header */}
      <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between flex-wrap gap-4">
            <div className="flex items-center gap-4">
              <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                  Outage Monitor Bot
                </h1>
                {health?.version && (
                  <span className="text-xs text-gray-400 dark:text-gray-500">
                    v{health.version}
                  </span>
                )}
              </div>
              <HealthBadge health={health} isLoading={loading} />
            </div>
            <div className="flex items-center gap-2 flex-wrap justify-end">
              <ThemeToggle />
              <button
                onClick={loadData}
                disabled={loading}
                className="px-3 py-2 text-xs sm:text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
              >
                {loading ? 'Refreshing...' : 'Refresh'}
              </button>
              <button
                onClick={handleReload}
                disabled={reloading || !api.getApiKey()}
                className="px-3 py-2 text-xs sm:text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:opacity-50"
              >
                {reloading ? 'Reloading...' : 'Reload Bot'}
              </button>
              <button
                onClick={handleClearApiKey}
                className="px-3 py-2 text-xs sm:text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                <span className="hidden sm:inline">Change </span>API Key
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Tab Navigation */}
      <TabNavigation activeTab={activeTab} onTabChange={setActiveTab} />

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {error && (
          <div className="mb-6 bg-error-50 dark:bg-error-900/30 border border-error-200 dark:border-error-700 rounded-lg p-4">
            <p className="text-sm text-error-700 dark:text-error-400">
              <strong>Error:</strong> {error}
            </p>
          </div>
        )}

        {/* Status & Config Tab */}
        {activeTab === 'status' && (
          <div className="space-y-8">
            {/* Status Cards Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-6">
              <StatusCard
                title="System Uptime"
                value={health ? formatUptime(health.uptime_seconds) : '−'}
                description="Total running time"
                icon={<span className="text-2xl">⏱️</span>}
              />
              <StatusCard
                title="Monitor Status"
                value={
                  health?.monitor_running ? (
                    <span className="text-success-600">Active</span>
                  ) : (
                    <span className="text-error-600">Inactive</span>
                  )
                }
                description={health?.monitor_running ? 'Checking sources' : 'Not running'}
                icon={<span className="text-2xl">🔍</span>}
              />
              <StatusCard
                title="Telegram Status"
                value={
                  health?.telegram_connected ? (
                    <span className="text-success-600">Connected</span>
                  ) : status?.bot.web_only_mode ? (
                    <span className="text-gray-500 dark:text-gray-400">Not Configured</span>
                  ) : (
                    <span className="text-error-600">Disconnected</span>
                  )
                }
                description={
                  health?.telegram_connected
                    ? 'Notifications enabled'
                    : status?.bot.web_only_mode
                    ? 'Web-only mode'
                    : 'Check token'
                }
                icon={<span className="text-2xl">✈️</span>}
              />
              <StatusCard
                title="Active Sources"
                value={status?.bot.active_sources ?? '−'}
                description={`${status?.bot.total_sources ?? 0} total sources`}
                icon={<span className="text-2xl">📡</span>}
              />
              <StatusCard
                title="API Status"
                value={
                  health?.api_running ? (
                    <span className="text-success-600">Online</span>
                  ) : (
                    <span className="text-error-600">Offline</span>
                  )
                }
                description={status ? `Port ${status.api.port}` : ''}
                icon={<span className="text-2xl">🌐</span>}
              />
            </div>

            {/* Two Column Layout */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              {/* Configuration Panel (2 columns) */}
              <div className="lg:col-span-2">
                <ConfigPanel
                  config={config}
                  onUpdate={handleConfigUpdate}
                  isLoading={loading}
                />
              </div>

              {/* Right Sidebar */}
              <div className="space-y-6">
                {/* Auto-Restart Info */}
                {status?.bot.auto_restart && (
                  <AutoRestartInfo info={status.bot.auto_restart} />
                )}

                {/* Bot Details */}
                <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Bot Details</h3>
                  <div className="space-y-3">
                    {status?.bot.started_at && (
                      <div>
                        <p className="text-xs text-gray-500 dark:text-gray-400">Started At</p>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {new Date(status.bot.started_at).toLocaleString()}
                        </p>
                      </div>
                    )}
                    {status?.bot.uptime && (
                      <div>
                        <p className="text-xs text-gray-500 dark:text-gray-400">Uptime</p>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{status.bot.uptime}</p>
                      </div>
                    )}
                    {status?.bot.last_error && (
                      <div>
                        <p className="text-xs text-gray-500 dark:text-gray-400">Last Error</p>
                        <p className="text-sm font-medium text-error-600">
                          {status.bot.last_error}
                        </p>
                      </div>
                    )}
                  </div>
                </div>

                {/* System Info */}
                {status?.system && (
                  <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">System Info</h3>
                    <div className="space-y-3">
                      <div>
                        <p className="text-xs text-gray-500 dark:text-gray-400">Started At</p>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {new Date(status.system.started_at).toLocaleString()}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-500 dark:text-gray-400">Total Uptime</p>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {formatUptime(status.system.uptime_seconds)}
                        </p>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Sources Tab */}
        {activeTab === 'sources' && (
          <SourcesPanel
            sources={sources}
            config={config}
            onCreateSource={handleCreateSource}
            onUpdateSource={handleUpdateSource}
            onDeleteSource={handleDeleteSource}
            onPauseSource={handlePauseSource}
            onResumeSource={handleResumeSource}
            isLoading={loading}
          />
        )}

        {/* Sinks Tab */}
        {activeTab === 'sinks' && <SinksPanel onToast={addToast} />}

        {/* Events Tab */}
        {activeTab === 'events' && <EventsPanel sources={sources} />}
      </main>
    </div>
  )
}

export default App
