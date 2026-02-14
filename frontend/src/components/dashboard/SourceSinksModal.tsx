import { useState, useEffect } from 'react'
import { api } from '../../lib/api'
import type { Source, Webhook, TelegramChat } from '../../types'

interface SourceSinksModalProps {
  source: Source
  isOpen: boolean
  onClose: () => void
  onSinksUpdated: () => void
}

export function SourceSinksModal({
  source,
  isOpen,
  onClose,
  onSinksUpdated,
}: SourceSinksModalProps) {
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [telegramChats, setTelegramChats] = useState<TelegramChat[]>([])
  const [selectedWebhookIds, setSelectedWebhookIds] = useState<Set<string>>(new Set())
  const [selectedChatIds, setSelectedChatIds] = useState<Set<number>>(new Set())
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [updating, setUpdating] = useState(false)

  useEffect(() => {
    if (isOpen) {
      loadSinks()
    }
  }, [isOpen])

  const loadSinks = async () => {
    setLoading(true)
    setError(null)
    try {
      const [hooks, chats, sourceHooks] = await Promise.all([
        api.getWebhooks(),
        api.getTelegramChats(),
        api.getSourceWebhooks(source.id),
      ])
      setWebhooks(hooks || [])
      setTelegramChats(chats || [])

      // Set currently selected webhooks
      const selectedHooks = new Set<string>()
      if (sourceHooks) {
        sourceHooks.forEach((hook) => selectedHooks.add(hook.id))
      }
      setSelectedWebhookIds(selectedHooks)

      // TODO: Load current telegram chat associations
      // For now, start with empty selections
      setSelectedChatIds(new Set())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sinks')
    } finally {
      setLoading(false)
    }
  }

  const toggleWebhook = async (webhookId: string) => {
    const newSelected = new Set(selectedWebhookIds)
    const isAdding = !newSelected.has(webhookId)

    setUpdating(true)
    try {
      if (isAdding) {
        await api.addSourceWebhook(source.id, webhookId)
        newSelected.add(webhookId)
      } else {
        await api.removeSourceWebhook(source.id, webhookId)
        newSelected.delete(webhookId)
      }
      setSelectedWebhookIds(newSelected)
      // Don't close modal - let user select more sinks
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update webhook association')
    } finally {
      setUpdating(false)
    }
  }

  const toggleTelegramChat = async (chatId: number) => {
    const newSelected = new Set(selectedChatIds)
    const isAdding = !newSelected.has(chatId)

    setUpdating(true)
    try {
      if (isAdding) {
        await api.addSourceTelegramChat(source.id, chatId)
        newSelected.add(chatId)
      } else {
        await api.removeSourceTelegramChat(source.id, chatId)
        newSelected.delete(chatId)
      }
      setSelectedChatIds(newSelected)
      // Don't close modal - let user select more sinks
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update telegram chat association')
    } finally {
      setUpdating(false)
    }
  }

  const handleClose = () => {
    onSinksUpdated()
    onClose()
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-lg max-w-md w-full">
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">
            Configure Sinks for "{source.name}"
          </h2>
        </div>

        <div className="px-6 py-4 max-h-96 overflow-y-auto">
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          {loading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="animate-pulse h-8 bg-gray-200 rounded"></div>
              ))}
            </div>
          ) : (
            <>
              {/* Webhooks Section */}
              {webhooks.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-sm font-semibold text-gray-900 mb-3">Webhooks</h3>
                  <div className="space-y-2">
                    {webhooks.map((webhook) => (
                      <label
                        key={webhook.id}
                        className="flex items-center gap-3 p-2 rounded hover:bg-gray-50 cursor-pointer"
                      >
                        <input
                          type="checkbox"
                          checked={selectedWebhookIds.has(webhook.id)}
                          onChange={() => toggleWebhook(webhook.id)}
                          disabled={updating}
                          className="w-4 h-4 text-primary-600 rounded"
                        />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-gray-900 truncate">{webhook.url}</p>
                          <p className="text-xs text-gray-500">
                            {webhook.method} {webhook.enabled ? '✓' : '(disabled)'}
                          </p>
                        </div>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              {/* Telegram Chats Section */}
              {telegramChats.length > 0 && (
                <div>
                  <h3 className="text-sm font-semibold text-gray-900 mb-3">Telegram Chats</h3>
                  <div className="space-y-2">
                    {telegramChats.map((chat) => (
                      <label
                        key={chat.chat_id}
                        className="flex items-center gap-3 p-2 rounded hover:bg-gray-50 cursor-pointer"
                      >
                        <input
                          type="checkbox"
                          checked={selectedChatIds.has(chat.chat_id)}
                          onChange={() => toggleTelegramChat(chat.chat_id)}
                          disabled={updating}
                          className="w-4 h-4 text-primary-600 rounded"
                        />
                        <div className="flex-1">
                          <p className="text-sm text-gray-900">Chat ID: {chat.chat_id}</p>
                          <p className="text-xs text-gray-500">
                            Added {new Date(chat.created_at).toLocaleDateString()}
                          </p>
                        </div>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              {webhooks.length === 0 && telegramChats.length === 0 && (
                <p className="text-sm text-gray-500 text-center py-4">
                  No sinks available. Create webhooks or telegram chats first.
                </p>
              )}
            </>
          )}
        </div>

        <div className="border-t border-gray-200 px-6 py-4 flex gap-2 justify-end">
          <button
            onClick={handleClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  )
}
