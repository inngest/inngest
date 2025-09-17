export const parseCelSearchError = (error: Error | null | undefined) => {
  if (!error) return undefined;

  // If the error has graphQLErrors (similar to urql structure)
  if ('graphQLErrors' in error && Array.isArray(error.graphQLErrors)) {
    return error.graphQLErrors.find(
      (gqlError: any) => gqlError.extensions?.code === 'expression_invalid'
    );
  }

  // If it's a standard error with message containing CEL validation info
  if (
    error.message &&
    (error.message.includes('expression_invalid') ||
      error.message.includes('invalid CEL expression'))
  ) {
    return error;
  }

  return undefined;
};
