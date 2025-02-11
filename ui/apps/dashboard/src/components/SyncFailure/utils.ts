import { isErrorCode, parseErrorData, type CodedError, type ErrorCode } from '@/codedError';

const messages = {
  account_mismatch: "The app's signing key is for the wrong account.",
  app_mismatch: 'The app at the provided URL does not match the app you are trying to sync.',
  app_uninitialized: 'Do an initial sync before resyncing.',
  batch_size_too_large: 'Configured batch size is too large',
  env_archived: 'Cannot sync an app to an archived environment.',
  env_mismatch: "The app's signing key is for the wrong environment.",
  env_unspecified:
    "The app's signing key is for a branch environment but the app did not specify a branch environment name.",
  forbidden: 'Forbidden response from URL.',
  host_private: "The app's reported host is private (e.g. localhost).",
  http_bad_request: 'Bad request response from URL.',
  http_forbidden: 'Forbidden response from URL.',
  http_internal_server_error: 'Internal server error response from URL.',
  http_method_not_allowed: 'Method not allowed response from URL.',
  http_not_found: 'Not found response from URL.',
  http_unauthorized: 'Unauthorized response from URL.',
  http_unreachable: 'The URL is unreachable.',
  http_unsupported_protocol: 'The provided URL uses an unsupported protocol.',
  internal_server_error: 'Internal server error response from URL.',
  invalid_function: 'A function is invalid.',
  invalid_signing_key: "The app's signing key is invalid.",
  missing_branch_env_name: 'Branch environment name not specified.',
  missing_signing_key: 'The app is not using a signing key.',
  no_functions: 'No functions found in the app.',
  not_sdk: 'The URL is not hosting an Inngest SDK',
  response_not_signed: 'SDK response was not signed. Is it in dev mode?',
  server_kind_mismatch: 'The app is not in cloud mode',
  sig_verification_failed:
    'Signature verification failed. Is your app using the correct signing key?',
  signing_key_invalid: "The app's signing key is invalid.",
  signing_key_unspecified: 'The app is not using a signing key.',
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

  const data = parseErrorData(error.data);
  if (data) {
    if ('errors' in data) {
      // We're dealing with a "multi-error" (probably a config error). We'll
      // return the unique messages for all the errors

      const set = new Set();
      for (const err of data.errors) {
        if (err.message) {
          set.add(err.message);
        }
      }

      if (set.size > 0) {
        return Array.from(set).join('\n');
      }
    }
  }

  return 'Something went wrong.';
}
