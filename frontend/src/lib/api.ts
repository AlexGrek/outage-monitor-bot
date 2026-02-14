import type {
  HealthResponse,
  StatusResponse,
  ConfigResponse,
  ConfigEntry,
  UpdateConfigRequest,
  UpdateConfigResponse,
  ReloadResponse,
  Source,
  CreateSourceRequest,
  UpdateSourceRequest,
  Webhook,
  CreateWebhookRequest,
  UpdateWebhookRequest,
  TelegramChat,
  StatusChangeEvent,
} from '../types'

const API_BASE = '/api'

class ApiClient {
  private apiKey: string = ''

  setApiKey(key: string) {
    this.apiKey = key
    if (typeof window !== 'undefined') {
      localStorage.setItem('api_key', key)
    }
  }

  getApiKey(): string {
    if (!this.apiKey && typeof window !== 'undefined') {
      this.apiKey = localStorage.getItem('api_key') || ''
    }
    return this.apiKey
  }

  clearApiKey() {
    this.apiKey = ''
    if (typeof window !== 'undefined') {
      localStorage.removeItem('api_key')
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    }

    // Add API key for authenticated endpoints
    if (!endpoint.includes('/health') && this.getApiKey()) {
      headers['X-API-Key'] = this.getApiKey()
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({
        error: response.statusText,
      }))
      throw new Error(error.error || `HTTP ${response.status}`)
    }

    return response.json()
  }

  // Health endpoint (no auth required)
  async getHealth(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health')
  }

  // Status endpoint (requires auth)
  async getStatus(): Promise<StatusResponse> {
    return this.request<StatusResponse>('/status')
  }

  // Config endpoints (require auth)
  async getAllConfig(): Promise<ConfigResponse> {
    return this.request<ConfigResponse>('/config')
  }

  async getConfig(key: string): Promise<ConfigEntry> {
    return this.request<ConfigEntry>(`/config/${key}`)
  }

  async updateConfig(
    key: string,
    value: string
  ): Promise<UpdateConfigResponse> {
    return this.request<UpdateConfigResponse>(`/config/${key}`, {
      method: 'PUT',
      body: JSON.stringify({ value } as UpdateConfigRequest),
    })
  }

  async reloadBot(): Promise<ReloadResponse> {
    return this.request<ReloadResponse>('/config/reload', {
      method: 'POST',
    })
  }

  // Source endpoints (require auth)
  async getSources(): Promise<Source[]> {
    return this.request<Source[]>('/sources')
  }

  async createSource(data: CreateSourceRequest): Promise<Source> {
    return this.request<Source>('/sources', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateSource(id: string, data: UpdateSourceRequest): Promise<Source> {
    return this.request<Source>(`/sources/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteSource(id: string): Promise<{ message: string; id: string }> {
    return this.request<{ message: string; id: string }>(`/sources/${id}`, {
      method: 'DELETE',
    })
  }

  async pauseSource(id: string): Promise<{ message: string; id: string }> {
    return this.request<{ message: string; id: string }>(
      `/sources/${id}/pause`,
      { method: 'POST' }
    )
  }

  async resumeSource(id: string): Promise<{ message: string; id: string }> {
    return this.request<{ message: string; id: string }>(
      `/sources/${id}/resume`,
      { method: 'POST' }
    )
  }

  // Webhook endpoints (require auth)
  async getWebhooks(): Promise<Webhook[]> {
    return this.request<Webhook[]>('/webhooks')
  }

  async createWebhook(data: CreateWebhookRequest): Promise<Webhook> {
    return this.request<Webhook>('/webhooks', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateWebhook(id: string, data: UpdateWebhookRequest): Promise<Webhook> {
    return this.request<Webhook>(`/webhooks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteWebhook(id: string): Promise<{ message: string; id: string }> {
    return this.request<{ message: string; id: string }>(`/webhooks/${id}`, {
      method: 'DELETE',
    })
  }

  // Telegram chat endpoints (require auth)
  async getTelegramChats(): Promise<TelegramChat[]> {
    return this.request<TelegramChat[]>('/telegram-chats')
  }

  async addTelegramChat(chatId: number): Promise<TelegramChat> {
    return this.request<TelegramChat>('/telegram-chats', {
      method: 'POST',
      body: JSON.stringify({ chat_id: chatId }),
    })
  }

  async removeTelegramChat(chatId: number): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/telegram-chats/${chatId}`, {
      method: 'DELETE',
    })
  }

  // Status change events endpoint (require auth)
  async getStatusChangeEvents(filters?: {
    source_id?: string
    limit?: number
    offset?: number
  }): Promise<StatusChangeEvent[]> {
    const params = new URLSearchParams()
    if (filters?.source_id) params.append('source_id', filters.source_id)
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const query = params.toString()
    const endpoint = query ? `/events?${query}` : '/events'
    return this.request<StatusChangeEvent[]>(endpoint)
  }
}

export const api = new ApiClient()
