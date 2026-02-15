import { useState, useCallback, useEffect } from 'react'
import { api } from '../../lib/api'
import type { Webhook, CreateWebhookRequest, TelegramChat } from '../../types'
import type { ToastMessage } from './Toast'

interface SinksPanelProps {
  onToast: (toast: Omit<ToastMessage, 'id'>) => void
}

export function SinksPanel({ onToast }: SinksPanelProps) {
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [telegramChats, setTelegramChats] = useState<TelegramChat[]>([])
  const [error, setError] = useState<string | null>(null)

  // Webhook management
  const [showWebhookForm, setShowWebhookForm] = useState(false)
  const [webhookForm, setWebhookForm] = useState<CreateWebhookRequest>({
    name: '',
    url: '',
    method: 'POST',
    enabled: true,
  })
  const [submittingWebhook, setSubmittingWebhook] = useState(false)

  // Telegram management
  const [showChatForm, setShowChatForm] = useState(false)
  const [chatIdInput, setChatIdInput] = useState('')
  const [chatNameInput, setChatNameInput] = useState('')
  const [submittingChat, setSubmittingChat] = useState(false)

  const inputClasses = "w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-primary-500 focus:border-primary-500"

  const loadData = useCallback(async () => {
    try {
      setError(null)
      const [webhooksData, chatsData] = await Promise.all([
        api.getWebhooks(),
        api.getTelegramChats(),
      ])
      setWebhooks(webhooksData)
      setTelegramChats(chatsData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sinks')
    }
  }, [])

  useEffect(() => {
    if (api.getApiKey()) {
      loadData()
    }
  }, [loadData])

  const handleCreateWebhook = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      setSubmittingWebhook(true)
      setError(null)
      await api.createWebhook(webhookForm)
      setWebhookForm({ name: '', url: '', method: 'POST', enabled: true })
      setShowWebhookForm(false)
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create webhook')
    } finally {
      setSubmittingWebhook(false)
    }
  }

  const handleDeleteWebhook = async (id: string) => {
    if (!window.confirm('Delete this webhook?')) return
    try {
      setError(null)
      await api.deleteWebhook(id)
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete webhook')
    }
  }

  const handleAddChat = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      setSubmittingChat(true)
      setError(null)
      const chatId = parseInt(chatIdInput, 10)
      if (isNaN(chatId)) {
        throw new Error('Invalid chat ID')
      }
      await api.addTelegramChat(chatId, chatNameInput.trim() || undefined)
      setChatIdInput('')
      setChatNameInput('')
      setShowChatForm(false)
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add chat')
    } finally {
      setSubmittingChat(false)
    }
  }

  const handleRemoveChat = async (chatId: number, name?: string) => {
    const label = name ? `"${name}"` : String(chatId)
    if (!window.confirm(`Remove chat ${label}?`)) return
    try {
      setError(null)
      await api.removeTelegramChat(chatId)
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove chat')
    }
  }

  const handleTestTelegramChat = async (chatId: number) => {
    try {
      setError(null)
      await api.testTelegramChat(chatId)
      onToast({
        type: 'success',
        title: 'Test notification sent!',
        message: `Test message successfully sent to Telegram chat ${chatId}`,
      })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to send test notification'
      setError(errorMessage)
      onToast({
        type: 'error',
        title: 'Test notification failed',
        message: errorMessage,
      })
    }
  }

  const handleTestWebhook = async (webhookId: string, url: string) => {
    try {
      setError(null)
      await api.testWebhook(webhookId)
      onToast({
        type: 'success',
        title: 'Test webhook sent!',
        message: `Test payload successfully sent to ${url}`,
      })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to send test notification'
      setError(errorMessage)
      onToast({
        type: 'error',
        title: 'Test webhook failed',
        message: errorMessage,
      })
    }
  }

  if (!api.getApiKey()) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <p className="text-gray-500 dark:text-gray-400">Authenticate to manage sinks</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {error && (
        <div className="bg-error-50 dark:bg-error-900/30 border border-error-200 dark:border-error-700 rounded-lg p-4">
          <p className="text-sm text-error-700 dark:text-error-400">{error}</p>
        </div>
      )}

      {/* Telegram Chats Section */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Telegram Chats</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Manage Telegram chat destinations
            </p>
          </div>
          {!showChatForm && (
            <button
              onClick={() => setShowChatForm(true)}
              className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700"
            >
              + Add Chat
            </button>
          )}
        </div>

        {showChatForm && (
          <form onSubmit={handleAddChat} className="mb-6 p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Name
                </label>
                <input
                  type="text"
                  value={chatNameInput}
                  onChange={(e) => setChatNameInput(e.target.value)}
                  placeholder="e.g. Team Alerts"
                  className={inputClasses}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Chat ID
                </label>
                <input
                  type="number"
                  value={chatIdInput}
                  onChange={(e) => setChatIdInput(e.target.value)}
                  placeholder="Enter Telegram chat ID"
                  className={inputClasses}
                  required
                />
              </div>
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={submittingChat}
                  className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:opacity-50"
                >
                  {submittingChat ? 'Adding...' : 'Add'}
                </button>
                <button
                  type="button"
                  onClick={() => setShowChatForm(false)}
                  className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
                >
                  Cancel
                </button>
              </div>
            </div>
          </form>
        )}

        <div className="space-y-2">
          {telegramChats.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">No chats configured</p>
          ) : (
            telegramChats.map((chat) => (
              <div
                key={chat.chat_id}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-900 rounded-lg"
              >
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    {chat.name ? chat.name : `Chat ${chat.chat_id}`}
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    ID: {chat.chat_id}
                    {chat.created_at
                      ? ` · Added ${new Date(chat.created_at).toLocaleDateString()}`
                      : ''}
                  </p>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => handleTestTelegramChat(chat.chat_id)}
                    className="px-3 py-1 text-sm text-primary-600 hover:text-primary-700 font-medium"
                  >
                    Test
                  </button>
                  <button
                    onClick={() => handleRemoveChat(chat.chat_id, chat.name)}
                    className="px-3 py-1 text-sm text-error-600 hover:text-error-700 font-medium"
                  >
                    Remove
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Webhooks Section */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Webhooks</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Configure HTTP webhooks for status change notifications
            </p>
          </div>
          {!showWebhookForm && (
            <button
              onClick={() => setShowWebhookForm(true)}
              className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700"
            >
              + Create Webhook
            </button>
          )}
        </div>

        {showWebhookForm && (
          <form onSubmit={handleCreateWebhook} className="mb-6 p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Name
                </label>
                <input
                  type="text"
                  value={webhookForm.name ?? ''}
                  onChange={(e) =>
                    setWebhookForm({ ...webhookForm, name: e.target.value })
                  }
                  placeholder="e.g. Slack Alerts"
                  className={inputClasses}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  URL
                </label>
                <input
                  type="url"
                  value={webhookForm.url}
                  onChange={(e) =>
                    setWebhookForm({ ...webhookForm, url: e.target.value })
                  }
                  placeholder="https://example.com/webhook"
                  className={inputClasses}
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  HTTP Method
                </label>
                <select
                  value={webhookForm.method}
                  onChange={(e) =>
                    setWebhookForm({
                      ...webhookForm,
                      method: e.target.value as 'GET' | 'POST' | 'PUT',
                    })
                  }
                  className={inputClasses}
                >
                  <option value="GET">GET</option>
                  <option value="POST">POST</option>
                  <option value="PUT">PUT</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Headers (JSON) - Optional
                </label>
                <textarea
                  value={
                    webhookForm.headers
                      ? JSON.stringify(webhookForm.headers, null, 2)
                      : ''
                  }
                  onChange={(e) => {
                    try {
                      const headers = e.target.value
                        ? JSON.parse(e.target.value)
                        : undefined
                      setWebhookForm({ ...webhookForm, headers })
                    } catch {
                      // Invalid JSON, just update the text
                    }
                  }}
                  placeholder='{"Authorization": "Bearer token"}'
                  rows={3}
                  className={`${inputClasses} font-mono text-sm`}
                />
              </div>

              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="webhook-enabled"
                  checked={webhookForm.enabled}
                  onChange={(e) =>
                    setWebhookForm({ ...webhookForm, enabled: e.target.checked })
                  }
                  className="rounded border-gray-300 dark:border-gray-600"
                />
                <label htmlFor="webhook-enabled" className="text-sm text-gray-700 dark:text-gray-300">
                  Enabled
                </label>
              </div>

              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={submittingWebhook}
                  className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:opacity-50"
                >
                  {submittingWebhook ? 'Creating...' : 'Create'}
                </button>
                <button
                  type="button"
                  onClick={() => setShowWebhookForm(false)}
                  className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
                >
                  Cancel
                </button>
              </div>
            </div>
          </form>
        )}

        <div className="space-y-2">
          {webhooks.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">No webhooks configured</p>
          ) : (
            webhooks.map((webhook) => (
              <div
                key={webhook.id}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-900 rounded-lg"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {webhook.name ? webhook.name : webhook.url}
                    </p>
                    <span className="text-xs px-2 py-1 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded">
                      {webhook.method}
                    </span>
                    {webhook.enabled ? (
                      <span className="text-xs px-2 py-1 bg-success-100 dark:bg-success-900/40 text-success-700 dark:text-success-500 rounded">
                        Enabled
                      </span>
                    ) : (
                      <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded">
                        Disabled
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {webhook.name && webhook.url ? `${webhook.url} · ` : ''}
                    Created {new Date(webhook.created_at).toLocaleDateString()}
                    {webhook.last_triggered &&
                      ` • Last triggered ${new Date(webhook.last_triggered).toLocaleTimeString()}`}
                  </p>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => handleTestWebhook(webhook.id, webhook.url)}
                    disabled={!webhook.enabled}
                    className="px-3 py-1 text-sm text-primary-600 hover:text-primary-700 font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Test
                  </button>
                  <button
                    onClick={() => handleDeleteWebhook(webhook.id)}
                    className="px-3 py-1 text-sm text-error-600 hover:text-error-700 font-medium"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
