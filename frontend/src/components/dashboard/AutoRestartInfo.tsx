import type { AutoRestartInfo as AutoRestartInfoType } from '../../types'

interface AutoRestartInfoProps {
  info: AutoRestartInfoType | null
}

export function AutoRestartInfo({ info }: AutoRestartInfoProps) {
  if (!info) return null

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 shadow-sm">
      <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
        <span className="text-xl">🔄</span>
        Auto-Restart Status
      </h3>
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-sm text-gray-600">Enabled</span>
          <span className={`text-sm font-medium ${info.enabled ? 'text-success-600' : 'text-gray-500'}`}>
            {info.enabled ? 'Yes' : 'No'}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-gray-600">Attempts</span>
          <span className="text-sm font-medium text-gray-900">
            {info.attempts}
            {info.max_attempts > 0 && ` / ${info.max_attempts}`}
            {info.max_attempts === 0 && ' (unlimited)'}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-gray-600">Next Restart Delay</span>
          <span className="text-sm font-medium text-gray-900">{info.next_delay}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-gray-600">Timer Active</span>
          <span className={`text-sm font-medium ${info.timer_active ? 'text-warning-600' : 'text-gray-500'}`}>
            {info.timer_active ? 'Yes' : 'No'}
          </span>
        </div>
      </div>
    </div>
  )
}
