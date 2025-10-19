import { Button } from '@inngest/components/Button/Button'

type ExpandedRowProps = {
  eventName?: string
  payload?: string
  onReplay: (eventName: string, payload: string) => void
}

export function ExpandedRowActions({
  eventName,
  payload,
  onReplay,
}: ExpandedRowProps) {
  const isInternalEvent = eventName?.startsWith('inngest/')

  return (
    <div className="flex items-center gap-2">
      <Button
        label="Replay event"
        onClick={() => eventName && payload && onReplay(eventName, payload)}
        appearance="outlined"
        size="small"
        disabled={!eventName || isInternalEvent || !payload}
      />
    </div>
  )
}
