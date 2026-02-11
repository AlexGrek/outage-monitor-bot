import type { HealthResponse } from '../../types'

interface HealthBadgeProps {
  health: HealthResponse | null
  isLoading?: boolean
}

export function HealthBadge({ health, isLoading }: HealthBadgeProps) {
  if (isLoading || !health) {
    return (
      <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 text-gray-600">
        <div className="w-2 h-2 rounded-full bg-gray-400 animate-pulse" />
        <span className="text-sm font-medium">Loading...</span>
      </div>
    )
  }

  const statusConfig = {
    healthy: {
      bg: 'bg-success-50',
      text: 'text-success-700',
      dot: 'bg-success-500',
      label: 'Healthy',
    },
    unhealthy: {
      bg: 'bg-error-50',
      text: 'text-error-700',
      dot: 'bg-error-500',
      label: 'Unhealthy',
    },
    degraded: {
      bg: 'bg-warning-50',
      text: 'text-warning-700',
      dot: 'bg-warning-500',
      label: 'Degraded',
    },
  }

  const config = statusConfig[health.status]

  return (
    <div className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-full ${config.bg} ${config.text}`}>
      <div className={`w-2 h-2 rounded-full ${config.dot} animate-pulse`} />
      <span className="text-sm font-medium">{config.label}</span>
    </div>
  )
}
