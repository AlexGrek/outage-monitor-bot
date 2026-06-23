import { useState, useCallback, useEffect } from 'react'
import { api } from '../../lib/api'
import type { Countdown, CountdownRequest, TelegramChat, Webhook } from '../../types'
import type { ToastMessage } from './Toast'

interface CountdownsPanelProps {
  onToast: (toast: Omit<ToastMessage, 'id'>) => void
}

interface FormState {
  name: string
  target_date: string
  notify_time: string
  timezone: string
  message_template: string
  enabled: boolean
  chat_ids: number[]
  webhook_ids: string[]
}

const browserTimezone =
  typeof Intl !== 'undefined' ? Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC' : 'UTC'

// A handful of common zones for the picker; users can type any IANA name.
const COMMON_TIMEZONES = [
  'UTC',
  'Europe/Kyiv',
  'Europe/London',
  'Europe/Berlin',
  'Europe/Moscow',
  'America/New_York',
  'America/Chicago',
  'America/Los_Angeles',
  'America/Sao_Paulo',
  'Asia/Dubai',
  'Asia/Kolkata',
  'Asia/Shanghai',
  'Asia/Tokyo',
  'Australia/Sydney',
]

const emptyForm = (): FormState => ({
  name: '',
  target_date: '',
  notify_time: '09:00',
  timezone: browserTimezone,
  message_template: '{} days left',
  enabled: true,
  chat_ids: [],
  webhook_ids: [],
})

// daysUntil approximates the remaining whole days for display only. The server
// computes the authoritative value in the countdown's own timezone.
function daysUntil(targetDate: string): number | null {
  if (!targetDate) return null
  const parts = targetDate.split('-').map(Number)
  if (parts.length !== 3 || parts.some(isNaN)) return null
  const [y, m, d] = parts
  const target = new Date(y, m - 1, d)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  return Math.round((target.getTime() - today.getTime()) / 86_400_000)
}

const inputClasses =
  'w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500 text-base'

export function CountdownsPanel({ onToast }: CountdownsPanelProps) {
  const [countdowns, setCountdowns] = useState<Countdown[]>([])
  const [telegramChats, setTelegramChats] = useState<TelegramChat[]>([])
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [error, setError] = useState<string | null>(null)

  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm())
  const [submitting, setSubmitting] = useState(false)

  const loadData = useCallback(async () => {
    try {
      setError(null)
      const [countdownsData, chatsData, webhooksData] = await Promise.all([
        api.getCountdowns(),
        api.getTelegramChats(),
        api.getWebhooks(),
      ])
      setCountdowns(countdownsData)
      setTelegramChats(chatsData)
      setWebhooks(webhooksData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load countdowns')
    }
  }, [])

  useEffect(() => {
    if (api.getApiKey()) {
      loadData()
    }
  }, [loadData])

  const openCreate = () => {
    setEditingId(null)
    setForm(emptyForm())
    setShowForm(true)
  }

  const openEdit = (c: Countdown) => {
    setEditingId(c.id)
    setForm({
      name: c.name,
      target_date: c.target_date,
      notify_time: c.notify_time,
      timezone: c.timezone,
      message_template: c.message_template,
      enabled: c.enabled,
      chat_ids: c.chat_ids ?? [],
      webhook_ids: c.webhook_ids ?? [],
    })
    setShowForm(true)
  }

  const closeForm = () => {
    setShowForm(false)
    setEditingId(null)
  }

  const toggleChat = (chatId: number) => {
    setForm((prev) => ({
      ...prev,
      chat_ids: prev.chat_ids.includes(chatId)
        ? prev.chat_ids.filter((id) => id !== chatId)
        : [...prev.chat_ids, chatId],
    }))
  }

  const toggleWebhook = (webhookId: string) => {
    setForm((prev) => ({
      ...prev,
      webhook_ids: prev.webhook_ids.includes(webhookId)
        ? prev.webhook_ids.filter((id) => id !== webhookId)
        : [...prev.webhook_ids, webhookId],
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      setSubmitting(true)
      setError(null)
      const payload: CountdownRequest = { ...form }
      if (editingId) {
        await api.updateCountdown(editingId, payload)
      } else {
        await api.createCountdown(payload)
      }
      closeForm()
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save countdown')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (c: Countdown) => {
    if (!window.confirm(`Delete countdown "${c.name}"?`)) return
    try {
      setError(null)
      await api.deleteCountdown(c.id)
      if (editingId === c.id) closeForm()
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete countdown')
    }
  }

  const handleTest = async (c: Countdown) => {
    try {
      setError(null)
      const res = await api.testCountdown(c.id)
      onToast({
        type: 'success',
        title: 'Test notification sent!',
        message: `"${c.name}" sent to its sinks (${res.days_left} days left)`,
      })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to send test notification'
      setError(message)
      onToast({ type: 'error', title: 'Test failed', message })
    }
  }

  const sinkSummary = (c: Countdown): string => {
    const chats = c.chat_ids?.length ?? 0
    const hooks = c.webhook_ids?.length ?? 0
    const parts: string[] = []
    if (chats > 0) parts.push(`${chats} chat${chats > 1 ? 's' : ''}`)
    if (hooks > 0) parts.push(`${hooks} webhook${hooks > 1 ? 's' : ''}`)
    return parts.length > 0 ? parts.join(' · ') : 'No sinks'
  }

  if (!api.getApiKey()) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <p className="text-gray-500 dark:text-gray-400">Authenticate to manage countdowns</p>
      </div>
    )
  }

  const previewDays = daysUntil(form.target_date)
  const previewMessage =
    previewDays !== null
      ? form.message_template.replaceAll('{}', String(previewDays))
      : form.message_template

  return (
    <div className="space-y-6">
      {error && (
        <div className="bg-error-50 dark:bg-error-900/30 border border-error-200 dark:border-error-700 rounded-lg p-4">
          <p className="text-sm text-error-700 dark:text-error-400">{error}</p>
        </div>
      )}

      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <div className="flex items-center justify-between mb-6 gap-3">
          <div className="min-w-0">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Countdowns</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Daily “days until a date” notifications. Use{' '}
              <code className="px-1 bg-gray-100 dark:bg-gray-700 rounded">{'{}'}</code> in the
              message for the number of days left.
            </p>
          </div>
          {!showForm && (
            <button
              onClick={openCreate}
              className="shrink-0 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700"
            >
              + New<span className="hidden sm:inline"> Countdown</span>
            </button>
          )}
        </div>

        {showForm && (
          <form
            onSubmit={handleSubmit}
            className="mb-6 p-4 bg-gray-50 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700"
          >
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Name
                </label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="e.g. New Year"
                  className={inputClasses}
                  required
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Target date
                  </label>
                  <input
                    type="date"
                    value={form.target_date}
                    onChange={(e) => setForm({ ...form, target_date: e.target.value })}
                    className={inputClasses}
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Notify time
                  </label>
                  <input
                    type="time"
                    value={form.notify_time}
                    onChange={(e) => setForm({ ...form, notify_time: e.target.value })}
                    className={inputClasses}
                    required
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Timezone
                </label>
                <input
                  type="text"
                  list="countdown-timezones"
                  value={form.timezone}
                  onChange={(e) => setForm({ ...form, timezone: e.target.value })}
                  placeholder="e.g. Europe/Kyiv"
                  className={inputClasses}
                  required
                />
                <datalist id="countdown-timezones">
                  {COMMON_TIMEZONES.map((tz) => (
                    <option key={tz} value={tz} />
                  ))}
                </datalist>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Message template
                </label>
                <textarea
                  value={form.message_template}
                  onChange={(e) => setForm({ ...form, message_template: e.target.value })}
                  placeholder="{} days left until the big day!"
                  rows={2}
                  className={inputClasses}
                  required
                />
                {form.message_template.includes('{}') && previewDays !== null && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Preview: <span className="italic">{previewMessage}</span>
                  </p>
                )}
              </div>

              {/* Sink selection */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Send to
                </label>
                {telegramChats.length === 0 && webhooks.length === 0 ? (
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    No sinks available. Add Telegram chats or webhooks in the Sinks tab first.
                  </p>
                ) : (
                  <div className="space-y-3">
                    {telegramChats.length > 0 && (
                      <div>
                        <p className="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500 mb-1">
                          Telegram chats
                        </p>
                        <div className="space-y-1 max-h-40 overflow-y-auto">
                          {telegramChats.map((chat) => (
                            <label
                              key={chat.chat_id}
                              className="flex items-center gap-3 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
                            >
                              <input
                                type="checkbox"
                                checked={form.chat_ids.includes(chat.chat_id)}
                                onChange={() => toggleChat(chat.chat_id)}
                                className="w-4 h-4 rounded border-gray-300 dark:border-gray-600"
                              />
                              <span className="text-sm text-gray-900 dark:text-gray-100 truncate">
                                {chat.name ? chat.name : `Chat ${chat.chat_id}`}
                              </span>
                            </label>
                          ))}
                        </div>
                      </div>
                    )}
                    {webhooks.length > 0 && (
                      <div>
                        <p className="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500 mb-1">
                          Webhooks
                        </p>
                        <div className="space-y-1 max-h-40 overflow-y-auto">
                          {webhooks.map((webhook) => (
                            <label
                              key={webhook.id}
                              className="flex items-center gap-3 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
                            >
                              <input
                                type="checkbox"
                                checked={form.webhook_ids.includes(webhook.id)}
                                onChange={() => toggleWebhook(webhook.id)}
                                className="w-4 h-4 rounded border-gray-300 dark:border-gray-600"
                              />
                              <span className="text-sm text-gray-900 dark:text-gray-100 truncate">
                                {webhook.name ? webhook.name : webhook.url}
                              </span>
                            </label>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>

              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="countdown-enabled"
                  checked={form.enabled}
                  onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                  className="rounded border-gray-300 dark:border-gray-600"
                />
                <label htmlFor="countdown-enabled" className="text-sm text-gray-700 dark:text-gray-300">
                  Enabled
                </label>
              </div>

              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={submitting}
                  className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:opacity-50"
                >
                  {submitting ? 'Saving...' : editingId ? 'Save changes' : 'Create'}
                </button>
                <button
                  type="button"
                  onClick={closeForm}
                  className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
                >
                  Cancel
                </button>
              </div>
            </div>
          </form>
        )}

        <div className="space-y-2 max-h-[50vh] sm:max-h-none overflow-y-auto sm:overflow-visible">
          {countdowns.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">No countdowns configured</p>
          ) : (
            countdowns.map((c) => {
              const days = daysUntil(c.target_date)
              return (
                <div
                  key={c.id}
                  className="flex flex-col sm:flex-row sm:items-center sm:justify-between p-3 bg-gray-50 dark:bg-gray-900 rounded-lg gap-2"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                        {c.name}
                      </p>
                      {days !== null && (
                        <span className="text-xs px-2 py-1 bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-400 rounded shrink-0">
                          {days < 0 ? `${Math.abs(days)} days ago` : `${days} days left`}
                        </span>
                      )}
                      {c.enabled ? (
                        <span className="text-xs px-2 py-1 bg-success-100 dark:bg-success-900/40 text-success-700 dark:text-success-500 rounded shrink-0">
                          Enabled
                        </span>
                      ) : (
                        <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded shrink-0">
                          Disabled
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 truncate">
                      {c.target_date} · daily at {c.notify_time} ({c.timezone}) · {sinkSummary(c)}
                    </p>
                  </div>
                  <div className="flex gap-1.5 shrink-0">
                    <button
                      onClick={() => handleTest(c)}
                      className="px-3 py-2 text-xs font-medium text-primary-600 hover:text-primary-700 hover:bg-primary-50 dark:hover:bg-primary-900/30 rounded-md"
                    >
                      Test
                    </button>
                    <button
                      onClick={() => openEdit(c)}
                      className="px-3 py-2 text-xs font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-md"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleDelete(c)}
                      className="px-3 py-2 text-xs font-medium text-error-600 hover:text-error-700 hover:bg-error-50 dark:hover:bg-error-900/30 rounded-md"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}
