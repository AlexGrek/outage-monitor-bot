export interface HealthResponse {
  status: 'healthy' | 'unhealthy' | 'degraded'
  bot_running: boolean
  bot_healthy: boolean
  monitor_running: boolean
  telegram_connected: boolean
  api_running: boolean
  uptime: string
  uptime_seconds: number
  last_error?: string
}

export interface AutoRestartInfo {
  enabled: boolean
  attempts: number
  max_attempts: number
  next_delay: string
  timer_active: boolean
}

export interface BotStatus {
  running: boolean
  healthy: boolean
  monitor_running: boolean
  telegram_connected: boolean
  web_only_mode: boolean
  last_error?: string
  started_at?: string
  uptime?: string
  total_sources?: number
  active_sources?: number
  auto_restart?: AutoRestartInfo
  config?: {
    telegram_token: string
    allowed_users: number[]
    check_interval: string
    ping_count: number
    ping_timeout: string
    http_timeout: string
  }
}

export interface ApiInfo {
  enabled: boolean
  port: number
  uptime: string
}

export interface SystemInfo {
  uptime: string
  uptime_seconds: number
  started_at: string
}

export interface StatusResponse {
  timestamp: string
  bot: BotStatus
  api: ApiInfo
  config: Record<string, string>
  system: SystemInfo
}

export interface ConfigResponse {
  [key: string]: string
}

export interface ConfigEntry {
  key: string
  value: string
  updated_at: string
  updated_by: string
}

export interface UpdateConfigRequest {
  value: string
}

export interface UpdateConfigResponse {
  message: string
  key: string
  restarting: string
}

export interface ReloadResponse {
  message: string
}

export interface Source {
  id: string
  name: string
  type: 'ping' | 'http'
  target: string
  check_interval: number // nanoseconds
  current_status: number // 1=online, 0=offline, -1=unknown
  last_check_time: string
  last_change_time: string
  enabled: boolean
  created_at: string
}

export interface CreateSourceRequest {
  name: string
  type: 'ping' | 'http'
  target: string
  check_interval: string // e.g. "30s", "1m"
}

export interface UpdateSourceRequest {
  name: string
  type: 'ping' | 'http'
  target: string
  check_interval: string
  enabled: boolean
}
