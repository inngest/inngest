import { SignOutButton } from '@clerk/tanstack-react-start'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/sign-out')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <div>
      <SignOutButton />
    </div>
  )
}
