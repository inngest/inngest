import { MenuItem as MenuItem } from '@inngest/components/Menu/NewMenuItem'
import { EventLogsIcon } from '@inngest/components/icons/sections/EventLogs'
import { RunsIcon } from '@inngest/components/icons/sections/Runs'

export default function Monitor({ collapsed }: { collapsed: boolean }) {
  return (
    <div className={`jusity-center mt-5 flex flex-col`}>
      {collapsed ? (
        <div className="border-subtle mx-auto mb-1 w-6 border-b" />
      ) : (
        <div className="text-muted leading-4.5 mb-1 text-xs font-medium">
          Monitor
        </div>
      )}
      <MenuItem
        href="/runs"
        collapsed={collapsed}
        text="Runs"
        icon={<RunsIcon className="h-[18px] w-[18px]" />}
      />
      <MenuItem
        href="/events"
        collapsed={collapsed}
        text="Events"
        icon={<EventLogsIcon className="h-[18px] w-[18px]" />}
      />
    </div>
  )
}
