import { useState, useEffect, useCallback } from 'react'
import { api } from '../../lib/api'
import type { StatusChangeEvent, Source } from '../../types'

interface EventsPanelProps {
  sources?: Source[] | null
  isLoading?: boolean
}

export function EventsPanel({ sources = null, isLoading = false }: EventsPanelProps) {
  const [events, setEvents] = useState<StatusChangeEvent[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedSourceId, setSelectedSourceId] = useState<string | undefined>()

  const loadEvents = useCallback(async () => {
    try {
      setError(null)
      setLoading(true)
      const eventsData = await api.getStatusChangeEvents({
        source_id: selectedSourceId,
        limit: 100,
      })
      setEvents(eventsData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load events')
    } finally {
      setLoading(false)
    }
  }, [selectedSourceId])

  useEffect(() => {
    if (api.getApiKey()) {
      loadEvents()
      // Refresh events every 5 seconds
      const interval = setInterval(loadEvents, 5000)
      return () => clearInterval(interval)
    }
  }, [loadEvents])

  const formatDuration = (durationMs: number): string => {
    const seconds = Math.floor(durationMs / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)

    if (days > 0) {
      return `${days}d ${hours % 24}h`
    }
    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`
    }
    if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`
    }
    return `${seconds}s`
  }

  const getStatusEmoji = (status: number): string => {
    return status === 1 ? '🟢' : '🔴'
  }

  if (!api.getApiKey()) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6 shadow-sm">
        <p className="text-gray-500">Authenticate to view events</p>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 shadow-sm">
      <div className="mb-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Status Change Timeline</h3>

        {/* Filter by source */}
        {sources && sources.length > 0 && (
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Filter by Source
            </label>
            <select
              value={selectedSourceId || ''}
              onChange={(e) => setSelectedSourceId(e.target.value || undefined)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="">All Sources</option>
              {sources.map((source) => (
                <option key={source.id} value={source.id}>
                  {source.name}
                </option>
              ))}
            </select>
          </div>
        )}

        {error && (
          <div className="bg-error-50 border border-error-200 rounded-lg p-4 mb-4">
            <p className="text-sm text-error-700">{error}</p>
          </div>
        )}
      </div>

      {/* Events List */}
      <div className="space-y-3">
        {loading ? (
          <div className="text-center py-8">
            <p className="text-gray-500">Loading events...</p>
          </div>
        ) : events.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-500">No status changes recorded</p>
          </div>
        ) : (
          events.map((event) => {
            const isOnline = event.new_status === 1
            const bgColor = isOnline ? 'bg-success-50' : 'bg-error-50'
            const borderColor = isOnline ? 'border-success-200' : 'border-error-200'
            const textColor = isOnline ? 'text-success-700' : 'text-error-700'

            return (
              <div
                key={event.id}
                className={`border rounded-lg p-4 ${bgColor} ${borderColor}`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <h4 className="font-semibold text-gray-900">{event.source_name}</h4>
                      <span className={`text-lg ${textColor}`}>
                        {getStatusEmoji(event.old_status)} →{' '}
                        {getStatusEmoji(event.new_status)}
                      </span>
                    </div>

                    <div className="text-sm text-gray-700 mb-2">
                      {isOnline ? '🟢 Restored' : '🔴 Outage Detected'}
                      {event.duration_ms > 0 && (
                        <span className="ml-2 text-gray-600">
                          (was {isOnline ? 'offline' : 'online'} for{' '}
                          {formatDuration(event.duration_ms)})
                        </span>
                      )}
                    </div>

                    <p className="text-xs text-gray-500">
                      {new Date(event.timestamp).toLocaleString()}
                    </p>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
