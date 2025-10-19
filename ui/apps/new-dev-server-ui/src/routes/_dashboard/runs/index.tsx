import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_dashboard/runs/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/_dashboard/runs/"!</div>
}
