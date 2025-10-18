import { createFileRoute } from '@tanstack/react-router'
import { SignIn } from '@clerk/tanstack-react-start'

export const Route = createFileRoute('/sign-in/$')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <div className="flex flex-row justify-center items-center">
      <SignIn />
    </div>
  )
}
