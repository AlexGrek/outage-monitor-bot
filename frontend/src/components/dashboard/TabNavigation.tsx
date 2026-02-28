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
    <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-16 z-10">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex gap-1 sm:gap-2 overflow-x-auto scrollbar-none">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              className={`flex-shrink-0 px-3 sm:px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.id
                  ? 'border-primary-600 text-primary-600'
                  : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 hover:border-gray-300 dark:hover:border-gray-600'
              }`}
            >
              <span className="sm:mr-2">{tab.icon}</span>
              <span className="hidden sm:inline">{tab.label}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
