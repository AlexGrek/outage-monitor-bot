import { useState } from 'react'
import type { Source, CreateSourceRequest, UpdateSourceRequest } from '../../types'

interface SourcesPanelProps {
  sources: Source[] | null
  onCreateSource: (data: CreateSourceRequest) => Promise<void>
  onUpdateSource: (id: string, data: UpdateSourceRequest) => Promise<void>
  onDeleteSource: (id: string) => Promise<void>
  onPauseSource: (id: string) => Promise<void>
  onResumeSource: (id: string) => Promise<void>
  isLoading?: boolean
}

type SourceFormData = {
  name: string
  type: 'ping' | 'http'
  target: string
  check_interval: string
}

export function SourcesPanel({
  sources,
  onCreateSource,
  onUpdateSource,
  onDeleteSource,
  onPauseSource,
  onResumeSource,
  isLoading,
}: SourcesPanelProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingSource, setEditingSource] = useState<Source | null>(null)
  const [formData, setFormData] = useState<SourceFormData>({
    name: '',
    type: 'ping',
    target: '',
    check_interval: '30s',
  })
  const [submitting, setSubmitting] = useState(false)

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    try {
      await onCreateSource(formData)
      setShowCreateForm(false)
      setFormData({ name: '', type: 'ping', target: '', check_interval: '30s' })
    } catch (error) {
      console.error('Failed to create source:', error)
    } finally {
      setSubmitting(false)
    }
  }

  const handleEdit = (source: Source) => {
    setEditingSource(source)
    setFormData({
      name: source.name,
      type: source.type,
      target: source.target,
      check_interval: formatDuration(source.check_interval),
    })
  }

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingSource) return

    setSubmitting(true)
    try {
      await onUpdateSource(editingSource.id, {
        ...formData,
        enabled: editingSource.enabled,
      })
      setEditingSource(null)
      setFormData({ name: '', type: 'ping', target: '', check_interval: '30s' })
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
    return 'text-gray-500'
  }

  const getStatusText = (status: number) => {
    if (status === 1) return '🟢 Online'
    if (status === 0) return '🔴 Offline'
    return '⚪ Unknown'
  }

  if (isLoading || !sources) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Monitoring Sources</h3>
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="animate-pulse flex justify-between py-2">
              <div className="h-4 bg-gray-200 rounded w-1/3"></div>
              <div className="h-4 bg-gray-200 rounded w-1/4"></div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 shadow-sm">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">
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
          className="mb-6 p-4 bg-gray-50 rounded-lg border border-gray-200"
        >
          <h4 className="text-sm font-semibold text-gray-900 mb-3">
            {editingSource ? 'Edit Source' : 'New Source'}
          </h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Name
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="My Server"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Type
              </label>
              <select
                value={formData.type}
                onChange={(e) =>
                  setFormData({ ...formData, type: e.target.value as 'ping' | 'http' })
                }
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="ping">ICMP Ping</option>
                <option value="http">HTTP/HTTPS</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Target
              </label>
              <input
                type="text"
                value={formData.target}
                onChange={(e) => setFormData({ ...formData, target: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder={formData.type === 'ping' ? '8.8.8.8' : 'https://example.com'}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Check Interval
              </label>
              <input
                type="text"
                value={formData.check_interval}
                onChange={(e) =>
                  setFormData({ ...formData, check_interval: e.target.value })
                }
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="30s, 1m, 5m"
                required
              />
            </div>
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
                setFormData({ name: '', type: 'ping', target: '', check_interval: '30s' })
              }}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {/* Sources List */}
      <div className="space-y-2 max-h-96 overflow-y-auto">
        {sources.length === 0 ? (
          <p className="text-sm text-gray-500 text-center py-8">
            No sources yet. Click "Add Source" to create one.
          </p>
        ) : (
          sources.map((source) => (
            <div
              key={source.id}
              className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 border border-gray-100"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  <p className="text-sm font-medium text-gray-900">{source.name}</p>
                  <span className={`text-sm font-medium ${getStatusColor(source.current_status)}`}>
                    {getStatusText(source.current_status)}
                  </span>
                  {!source.enabled && (
                    <span className="text-xs px-2 py-0.5 bg-gray-200 text-gray-600 rounded">
                      Paused
                    </span>
                  )}
                </div>
                <p className="text-xs text-gray-500 mt-1">
                  {source.type.toUpperCase()} • {source.target} • every{' '}
                  {formatDuration(source.check_interval)}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleTogglePause(source)}
                  disabled={submitting}
                  className="px-3 py-1.5 text-xs font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
                >
                  {source.enabled ? 'Pause' : 'Resume'}
                </button>
                <button
                  onClick={() => handleEdit(source)}
                  disabled={submitting}
                  className="px-3 py-1.5 text-xs font-medium text-primary-600 hover:text-primary-700 hover:bg-primary-50 rounded-md"
                >
                  Edit
                </button>
                <button
                  onClick={() => handleDelete(source.id, source.name)}
                  disabled={submitting}
                  className="px-3 py-1.5 text-xs font-medium text-error-600 hover:text-error-700 hover:bg-error-50 rounded-md"
                >
                  Delete
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
