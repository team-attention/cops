import { List, Network } from 'lucide-react'
import { Tabs, TabsList, TabsTrigger } from '@/gen/shadcn/ui/tabs'
import type { ViewMode } from '../type/graph'

// Type guard to validate ViewMode values
const isViewMode = (value: string): value is ViewMode => {
  return value === 'chat' || value === 'graph'
}

interface ViewToggleProps {
  // Current view mode
  value: ViewMode
  // Callback when view mode changes
  onChange: (value: ViewMode) => void
}

// ViewToggle renders a tab-based toggle for switching between Chat and Graph views.
export const ViewToggle = ({ value, onChange }: ViewToggleProps) => {
  return (
    <Tabs
      value={value}
      onValueChange={(v) => isViewMode(v) && onChange(v)}
      className="w-fit"
    >
      <TabsList className="border border-zinc-800 bg-zinc-900/50">
        <TabsTrigger
          value="chat"
          className="gap-1.5 font-mono text-xs data-[state=active]:bg-zinc-800 data-[state=active]:text-cyan-400"
        >
          <List className="h-3.5 w-3.5" />
          Chat
        </TabsTrigger>
        <TabsTrigger
          value="graph"
          className="gap-1.5 font-mono text-xs data-[state=active]:bg-zinc-800 data-[state=active]:text-violet-400"
        >
          <Network className="h-3.5 w-3.5" />
          Graph
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )
}
