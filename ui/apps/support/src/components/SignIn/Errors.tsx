export const SignInRedirectErrors: Record<string, string> = {
  "session-expired": "Your session has expired. Please sign in again.",
  unauthorized: "You are not authorized to access this resource.",
  "invalid-token":
    "Your authentication token is invalid. Please sign in again.",
};

export function hasErrorMessage(
  error: string,
): error is keyof typeof SignInRedirectErrors {
  return error in SignInRedirectErrors;
}

export default SignInRedirectErrors;
