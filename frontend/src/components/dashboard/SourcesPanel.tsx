import { useState } from 'react'
import type { Source, CreateSourceRequest, UpdateSourceRequest } from '../../types'
import { SourceSinksModal } from './SourceSinksModal'

const GRACE_PERIOD_PRESETS = [1.1, 1.5, 2.0, 2.1, 2.5, 3.1, 4.1, 5, 10] as const
const DEFAULT_GRACE_MULTIPLIER = 2.5

interface SourcesPanelProps {
  sources: Source[] | null
  config?: Record<string, string> | null
  onCreateSource: (data: CreateSourceRequest) => Promise<void>
  onUpdateSource: (id: string, data: UpdateSourceRequest) => Promise<void>
  onDeleteSource: (id: string) => Promise<void>
  onPauseSource: (id: string) => Promise<void>
  onResumeSource: (id: string) => Promise<void>
  isLoading?: boolean
}

type SourceFormData = {
  name: string
  type: 'ping' | 'http' | 'webhook'
  target: string
  check_interval: string
  grace_period_multiplier: number
  grace_custom: string
  expected_headers: string
  expected_content: string
}

export function SourcesPanel({
  sources,
  config,
  onCreateSource,
  onUpdateSource,
  onDeleteSource,
  onPauseSource,
  onResumeSource,
  isLoading,
}: SourcesPanelProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingSource, setEditingSource] = useState<Source | null>(null)
  const [sinksSourceId, setSinksSourceId] = useState<string | null>(null)
  const [formData, setFormData] = useState<SourceFormData>({
    name: '',
    type: 'ping',
    target: '',
    check_interval: '30s',
    grace_period_multiplier: DEFAULT_GRACE_MULTIPLIER,
    grace_custom: '',
    expected_headers: '',
    expected_content: '',
  })
  const [submitting, setSubmitting] = useState(false)

  const webhookBaseUrl = config?.WEBHOOK_BASE_URL ?? config?.PUBLIC_URL ?? ''

  const buildCreatePayload = (): CreateSourceRequest => {
    const base = { name: formData.name, type: formData.type, check_interval: formData.check_interval }
    if (formData.type === 'webhook') {
      const mult = formData.grace_custom ? parseFloat(formData.grace_custom) : formData.grace_period_multiplier
      return {
        ...base,
        grace_period_multiplier: Number.isFinite(mult) ? mult : DEFAULT_GRACE_MULTIPLIER,
        expected_headers: formData.expected_headers || undefined,
        expected_content: formData.expected_content || undefined,
      }
    }
    return { ...base, target: formData.target }
  }

  const buildUpdatePayload = (): UpdateSourceRequest => {
    const base = {
      name: formData.name,
      type: formData.type,
      check_interval: formData.check_interval,
      enabled: editingSource!.enabled,
    }
    if (formData.type === 'webhook') {
      const mult = formData.grace_custom ? parseFloat(formData.grace_custom) : formData.grace_period_multiplier
      return {
        ...base,
        grace_period_multiplier: Number.isFinite(mult) ? mult : DEFAULT_GRACE_MULTIPLIER,
        expected_headers: formData.expected_headers || undefined,
        expected_content: formData.expected_content || undefined,
      }
    }
    return { ...base, target: formData.target }
  }

  const resetForm = () => {
    setFormData({
      name: '',
      type: 'ping',
      target: '',
      check_interval: '30s',
      grace_period_multiplier: DEFAULT_GRACE_MULTIPLIER,
      grace_custom: '',
      expected_headers: '',
      expected_content: '',
    })
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    try {
      await onCreateSource(buildCreatePayload())
      setShowCreateForm(false)
      resetForm()
    } catch (error) {
      console.error('Failed to create source:', error)
    } finally {
      setSubmitting(false)
    }
  }

  const handleEdit = (source: Source) => {
    setEditingSource(source)
    const grace = source.grace_period_multiplier ?? DEFAULT_GRACE_MULTIPLIER
    const isPreset = GRACE_PERIOD_PRESETS.includes(grace as (typeof GRACE_PERIOD_PRESETS)[number])
    setFormData({
      name: source.name,
      type: source.type,
      target: source.target ?? '',
      check_interval: formatDuration(source.check_interval),
      grace_period_multiplier: isPreset ? grace : DEFAULT_GRACE_MULTIPLIER,
      grace_custom: isPreset ? '' : String(grace),
      expected_headers: source.expected_headers ?? '',
      expected_content: source.expected_content ?? '',
    })
  }

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingSource) return

    setSubmitting(true)
    try {
      await onUpdateSource(editingSource.id, buildUpdatePayload())
      setEditingSource(null)
      resetForm()
    } catch (error) {
      console.error('Failed to update source:', error)
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete source "${name}"?`)) return

    setSubmitting(true)
    try {
      await onDeleteSource(id)
    } catch (error) {
      console.error('Failed to delete source:', error)
    } finally {
      setSubmitting(false)
    }
  }

  const handleTogglePause = async (source: Source) => {
    setSubmitting(true)
    try {
      if (source.enabled) {
        await onPauseSource(source.id)
      } else {
        await onResumeSource(source.id)
      }
    } catch (error) {
      console.error('Failed to toggle source:', error)
    } finally {
      setSubmitting(false)
    }
  }

  const formatDuration = (ns: number): string => {
    const seconds = ns / 1_000_000_000
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    return `${minutes}m`
  }

  const getStatusColor = (status: number) => {
    if (status === 1) return 'text-success-600'
    if (status === 0) return 'text-error-600'
    return 'text-gray-500 dark:text-gray-400'
  }

  const getStatusText = (status: number) => {
    if (status === 1) return '🟢 Online'
    if (status === 0) return '🔴 Offline'
    return '⚪ Unknown'
  }

  const inputClasses = "w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500"

  if (isLoading || !sources) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Monitoring Sources</h3>
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="animate-pulse flex justify-between py-2">
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3"></div>
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/4"></div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Monitoring Sources ({sources.length})
        </h3>
        {!showCreateForm && !editingSource && (
          <button
            onClick={() => setShowCreateForm(true)}
            className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700"
          >
            + Add Source
          </button>
        )}
      </div>

      {/* Create/Edit Form */}
      {(showCreateForm || editingSource) && (
        <form
          onSubmit={editingSource ? handleUpdate : handleCreate}
          className="mb-6 p-4 bg-gray-50 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700"
        >
          <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
            {editingSource ? 'Edit Source' : 'New Source'}
          </h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Name
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className={inputClasses}
                placeholder="My Server"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Type
              </label>
              <select
                value={formData.type}
                onChange={(e) =>
                  setFormData({ ...formData, type: e.target.value as 'ping' | 'http' | 'webhook' })
                }
                className={inputClasses}
              >
                <option value="ping">ICMP Ping</option>
                <option value="http">HTTP/HTTPS</option>
                <option value="webhook">Incoming Webhook</option>
              </select>
            </div>
            {formData.type !== 'webhook' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Target
                </label>
                <input
                  type="text"
                  value={formData.target}
                  onChange={(e) => setFormData({ ...formData, target: e.target.value })}
                  className={inputClasses}
                  placeholder={formData.type === 'ping' ? '8.8.8.8' : 'https://example.com'}
                  required
                />
              </div>
            )}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {formData.type === 'webhook' ? 'Expected period (heartbeat interval)' : 'Check Interval'}
              </label>
              <input
                type="text"
                value={formData.check_interval}
                onChange={(e) =>
                  setFormData({ ...formData, check_interval: e.target.value })
                }
                className={inputClasses}
                placeholder="30s, 1m, 5m"
                required
              />
            </div>
            {formData.type === 'webhook' && (
              <>
                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Grace period multiplier (mark offline if no heartbeat in period x this)
                  </label>
                  <div className="flex flex-wrap gap-2 items-center">
                    <select
                      value={formData.grace_custom ? 'custom' : String(formData.grace_period_multiplier)}
                      onChange={(e) => {
                        const v = e.target.value
                        if (v === 'custom') {
                          setFormData({ ...formData, grace_custom: String(DEFAULT_GRACE_MULTIPLIER), grace_period_multiplier: DEFAULT_GRACE_MULTIPLIER })
                        } else {
                          setFormData({ ...formData, grace_period_multiplier: parseFloat(v), grace_custom: '' })
                        }
                      }}
                      className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    >
                      {GRACE_PERIOD_PRESETS.map((n) => (
                        <option key={n} value={String(n)}>{n}x</option>
                      ))}
                      <option value="custom">Custom</option>
                    </select>
                    {formData.grace_custom !== '' && (
                      <input
                        type="number"
                        min={1}
                        max={100}
                        step={0.1}
                        value={formData.grace_custom}
                        onChange={(e) => setFormData({ ...formData, grace_custom: e.target.value })}
                        className="w-20 px-2 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
                        placeholder="e.g. 2.5"
                      />
                    )}
                  </div>
                </div>
                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Expected headers (optional, JSON) e.g. {`{"X-Auth":"secret"}`}
                  </label>
                  <input
                    type="text"
                    value={formData.expected_headers}
                    onChange={(e) => setFormData({ ...formData, expected_headers: e.target.value })}
                    className={`${inputClasses} font-mono text-sm`}
                    placeholder='{"X-Custom-Header": "value"}'
                  />
                </div>
                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Expected content in body (optional, substring)
                  </label>
                  <input
                    type="text"
                    value={formData.expected_content}
                    onChange={(e) => setFormData({ ...formData, expected_content: e.target.value })}
                    className={inputClasses}
                    placeholder="e.g. ok or heartbeat"
                  />
                </div>
              </>
            )}
          </div>
          <div className="flex gap-2 mt-4">
            <button
              type="submit"
              disabled={submitting}
              className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:opacity-50"
            >
              {submitting ? 'Saving...' : editingSource ? 'Update' : 'Create'}
            </button>
            <button
              type="button"
              onClick={() => {
                setShowCreateForm(false)
                setEditingSource(null)
                resetForm()
              }}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {/* Sources List */}
      <div className="space-y-2 max-h-96 overflow-y-auto">
        {sources.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-8">
            No sources yet. Click "Add Source" to create one.
          </p>
        ) : (
          sources.map((source) => (
            <div
              key={source.id}
              className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 border border-gray-100 dark:border-gray-700"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{source.name}</p>
                  <span className={`text-sm font-medium ${getStatusColor(source.current_status)}`}>
                    {getStatusText(source.current_status)}
                  </span>
                  {!source.enabled && (
                    <span className="text-xs px-2 py-0.5 bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
                      Paused
                    </span>
                  )}
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {source.type === 'webhook' ? (
                    <>
                      Webhook • every {formatDuration(source.check_interval)}
                      {source.webhook_token && (
                        <span className="block mt-1 font-mono text-gray-700 dark:text-gray-300 truncate" title={webhookBaseUrl ? `${webhookBaseUrl.replace(/\/$/, '')}/webhooks/incoming/${source.webhook_token}` : undefined}>
                          {webhookBaseUrl ? `${webhookBaseUrl.replace(/\/$/, '')}/webhooks/incoming/${source.webhook_token}` : `/webhooks/incoming/${source.webhook_token}`}
                        </span>
                      )}
                    </>
                  ) : (
                    <>
                      {source.type.toUpperCase()} • {source.target} • every {formatDuration(source.check_interval)}
                    </>
                  )}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setSinksSourceId(source.id)}
                  disabled={submitting}
                  className="px-3 py-1.5 text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded-md"
                  title="Configure webhooks and telegram chats for this source"
                >
                  Sinks
                </button>
                <button
                  onClick={() => handleTogglePause(source)}
                  disabled={submitting}
                  className="px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
                >
                  {source.enabled ? 'Pause' : 'Resume'}
                </button>
                <button
                  onClick={() => handleEdit(source)}
                  disabled={submitting}
                  className="px-3 py-1.5 text-xs font-medium text-primary-600 hover:text-primary-700 hover:bg-primary-50 dark:hover:bg-primary-900/30 rounded-md"
                >
                  Edit
                </button>
                <button
                  onClick={() => handleDelete(source.id, source.name)}
                  disabled={submitting}
                  className="px-3 py-1.5 text-xs font-medium text-error-600 hover:text-error-700 hover:bg-error-50 dark:hover:bg-error-900/30 rounded-md"
                >
                  Delete
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Sinks Configuration Modal */}
      {sinksSourceId && (
        <SourceSinksModal
          source={sources.find((s) => s.id === sinksSourceId)!}
          isOpen={!!sinksSourceId}
          onClose={() => setSinksSourceId(null)}
          onSinksUpdated={() => {
            // Modal has updated sinks, parent will refresh via normal polling
            setSinksSourceId(null)
          }}
        />
      )}
    </div>
  )
}
