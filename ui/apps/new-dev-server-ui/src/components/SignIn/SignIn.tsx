import { SignIn } from '@clerk/tanstack-react-start'

import SplitView from './SplitView'
import { useSearch } from '@tanstack/react-router'

const SignInRedirectErrors = {
  Unauthenticated: 'unauthenticated',
} as const

const signInRedirectErrorMessages = {
  [SignInRedirectErrors.Unauthenticated]:
    'Could not resume your session. Please sign in again.',
} as const

export default function SignInPage() {
  const searchParams = useSearch({ strict: false })

  function hasErrorMessage(
    error: string,
  ): error is keyof typeof signInRedirectErrorMessages {
    return error in signInRedirectErrorMessages
  }

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <SignIn
          appearance={{
            elements: {
              footer: 'bg-none',
            },
          }}
        />
        {/* {typeof searchParams.error === "string" && (
          <Alert severity="error" className="mx-auto max-w-xs">
            <p className="text-balance">
              {hasErrorMessage(searchParams.error)
                ? signInRedirectErrorMessages[searchParams.error]
                : searchParams.error}
            </p>
            <p className="mt-2">
              <Alert.Link
                size="medium"
                severity="error"
                href="/support"
                className="inline underline"
              >
                Contact support
              </Alert.Link>{" "}
              if this problem persists.
            </p>
          </Alert>
        )} */}
      </div>
    </SplitView>
  )
}
