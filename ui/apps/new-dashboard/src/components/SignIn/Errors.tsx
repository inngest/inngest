export const SignInRedirectErrors = {
  Unauthenticated: "unauthenticated",
} as const;

export type SignInRedirectError = keyof typeof SignInRedirectErrors;

export const hasErrorMessage = (
  error: string,
): error is keyof typeof SignInRedirectErrors => {
  return error in SignInRedirectErrors;
};

export default SignInRedirectErrors;
