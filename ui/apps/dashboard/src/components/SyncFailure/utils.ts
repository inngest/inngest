import { isErrorCode, type CodedError, type ErrorCode } from '@/codedError';

const messages = {
  app_mismatch: 'The app at the provided URL does not match the app you are trying to sync.',
  app_uninitialized: 'Do an initial sync before resyncing.',
  batch_size_too_large: 'Configured batch size is too large',
  env_archived: 'Cannot sync an app to an archived environment.',
  env_mismatch: "The app's signing key is for the wrong environment.",
  forbidden: 'Forbidden response from URL.',
  internal_server_error: 'Internal server error response from URL.',
  invalid_function: 'A function is invalid.',
  invalid_signing_key: "The app's signing key is invalid.",
  missing_branch_env_name: 'Branch environment name not specified.',
  missing_signing_key: 'The app is not using a signing key.',
  no_functions: 'No functions found in the app.',
  not_sdk: 'The URL is not hosting an Inngest SDK',
  too_many_pings: 'Too many requests to register in a short time window.',
  unauthorized: 'Unauthorized response from URL.',
  unreachable: 'The URL is unreachable.',
  unsupported_protocol: 'The provided URL uses an unsupported protocol.',
  url_invalid: 'The provided URL is not valid.',
  url_not_found: 'Not found response from URL.',
} as const satisfies { [key in Exclude<ErrorCode, 'unknown'>]: string };

export function getMessage(error: CodedError) {
  if (isErrorCode(error.code) && error.code !== 'unknown') {
    return messages[error.code];
  }

  if (error.message) {
    return error.message;
  }

  return 'Something went wrong.';
}
