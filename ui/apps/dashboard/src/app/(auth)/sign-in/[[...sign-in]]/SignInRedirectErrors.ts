const SignInRedirectErrors = {
  Unauthenticated: 'unauthenticated',
} as const;

export type SignInRedirectError = keyof typeof SignInRedirectErrors;

export default SignInRedirectErrors;
