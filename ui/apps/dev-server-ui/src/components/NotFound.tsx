import { RiErrorWarningLine } from '@remixicon/react'

export function NotFound() {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center">
      <div className="flex flex-row items-center gap-2">
        <RiErrorWarningLine className="h-6 w-6 text-subtle" />
        <h1 className="text-subtle text-lg">404 Page not found</h1>
      </div>
    </div>
  )
}
