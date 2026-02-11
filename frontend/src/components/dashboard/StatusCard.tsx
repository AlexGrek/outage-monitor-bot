import type { ReactNode } from 'react'

interface StatusCardProps {
  title: string
  value: string | number | ReactNode
  description?: string
  icon?: ReactNode
  trend?: 'up' | 'down' | 'neutral'
}

export function StatusCard({ title, value, description, icon, trend }: StatusCardProps) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 shadow-sm hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-gray-600">{title}</p>
          <div className="mt-2 flex items-baseline gap-2">
            <p className="text-2xl font-semibold text-gray-900">{value}</p>
            {trend && (
              <span
                className={`text-sm font-medium ${
                  trend === 'up'
                    ? 'text-success-600'
                    : trend === 'down'
                    ? 'text-error-600'
                    : 'text-gray-500'
                }`}
              >
                {trend === 'up' ? '↑' : trend === 'down' ? '↓' : '−'}
              </span>
            )}
          </div>
          {description && <p className="mt-1 text-sm text-gray-500">{description}</p>}
        </div>
        {icon && (
          <div className="flex-shrink-0 w-10 h-10 bg-primary-50 rounded-lg flex items-center justify-center text-primary-600">
            {icon}
          </div>
        )}
      </div>
    </div>
  )
}
