import type { ReactNode } from 'react'

export type TabId = 'status' | 'sources' | 'sinks' | 'events'

interface Tab {
  id: TabId
  label: string
  icon: ReactNode
}

const TABS: Tab[] = [
  { id: 'status', label: 'Status & Config', icon: '⚙️' },
  { id: 'sources', label: 'Sources', icon: '📡' },
  { id: 'sinks', label: 'Sinks', icon: '📤' },
  { id: 'events', label: 'Events', icon: '📋' },
]

interface TabNavigationProps {
  activeTab: TabId
  onTabChange: (tabId: TabId) => void
}

export function TabNavigation({ activeTab, onTabChange }: TabNavigationProps) {
  return (
    <div className="bg-white border-b border-gray-200 sticky top-16 z-10">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex gap-2">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.id
                  ? 'border-primary-600 text-primary-600'
                  : 'border-transparent text-gray-600 hover:text-gray-900 hover:border-gray-300'
              }`}
            >
              <span className="mr-2">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
