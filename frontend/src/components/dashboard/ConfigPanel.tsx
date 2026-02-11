import { useState } from 'react'
import type { ConfigResponse } from '../../types'

interface ConfigPanelProps {
  config: ConfigResponse | null
  onUpdate: (key: string, value: string) => Promise<void>
  isLoading?: boolean
}

export function ConfigPanel({ config, onUpdate, isLoading }: ConfigPanelProps) {
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleEdit = (key: string, currentValue: string) => {
    setEditingKey(key)
    setEditValue(currentValue)
  }

  const handleCancel = () => {
    setEditingKey(null)
    setEditValue('')
  }

  const handleSave = async () => {
    if (!editingKey) return

    setSubmitting(true)
    try {
      await onUpdate(editingKey, editValue)
      setEditingKey(null)
      setEditValue('')
    } catch (error) {
      console.error('Failed to update config:', error)
    } finally {
      setSubmitting(false)
    }
  }

  if (isLoading || !config) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Configuration</h3>
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="animate-pulse flex justify-between py-2">
              <div className="h-4 bg-gray-200 rounded w-1/3"></div>
              <div className="h-4 bg-gray-200 rounded w-1/2"></div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  const entries = Object.entries(config).sort(([a], [b]) => a.localeCompare(b))

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 shadow-sm">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Configuration</h3>
      <div className="space-y-3 max-h-96 overflow-y-auto">
        {entries.map(([key, value]) => (
          <div
            key={key}
            className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-gray-50 transition-colors"
          >
            <div className="flex-1 min-w-0 mr-4">
              <p className="text-sm font-medium text-gray-900">{key}</p>
              {editingKey === key ? (
                <input
                  type="text"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  className="mt-1 w-full px-3 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
                  autoFocus
                />
              ) : (
                <p className="mt-1 text-sm text-gray-500 truncate">{value}</p>
              )}
            </div>
            <div className="flex items-center gap-2">
              {editingKey === key ? (
                <>
                  <button
                    onClick={handleSave}
                    disabled={submitting}
                    className="px-3 py-1.5 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:opacity-50"
                  >
                    {submitting ? 'Saving...' : 'Save'}
                  </button>
                  <button
                    onClick={handleCancel}
                    disabled={submitting}
                    className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                </>
              ) : (
                <button
                  onClick={() => handleEdit(key, value)}
                  className="px-3 py-1.5 text-sm font-medium text-primary-600 hover:text-primary-700 hover:bg-primary-50 rounded-md transition-colors"
                >
                  Edit
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
