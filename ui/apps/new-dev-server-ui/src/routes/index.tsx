import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/')({
  component: Home,
})

function Home() {
  return (
    <div className="flex flex-col items-center justify-start mt-6 gap-2">
      <h1>Index Route</h1>
      dev server coming soon...
    </div>
  )
}
