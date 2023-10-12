import { z } from 'zod';

import { put } from '@/queries/api';

const failedResponseBody = z.object({
  data: z.object({
    error_code: z.string(),
    response_headers: z.record(z.string(), z.array(z.string())).nullable(),
    response_status_code: z.number(),
  }),
  error: z.string(),
});

// The source of truth for these error codes are in
// pkg/applogic/sdkhandlers/registration.go (as of the writing of this comment).
// We should eventually codegen this array, rather than manually maintaining it.
export const registrationErrorCodes = [
  'batch_size_too_large',
  'forbidden',
  'internal_server_error',
  'invalid_function',
  'invalid_signing_key',
  'missing_branch_env_name',
  'missing_signing_key',
  'no_functions',
  'too_many_pings',
  'unauthorized',
  'unreachable',
  'unsupported_protocol',
  'url_not_found',
] as const;

export type RegistrationErrorCode = (typeof registrationErrorCodes)[number];
function isRegistrationErrorCode(value: unknown): value is RegistrationErrorCode {
  return registrationErrorCodes.includes(value as RegistrationErrorCode);
}

export type RegistrationFailure = {
  errorCode: RegistrationErrorCode | undefined;
  headers: Record<string, string[]>;
  statusCode: number | undefined;
};

export async function deployViaUrl(input: string): Promise<RegistrationFailure | undefined> {
  let url: URL;
  try {
    url = new URL(input);
  } catch (err) {
    throw new Error('Please enter a valid URL, e.g. https://example.com/api/inngest');
  }

  let res: Response;
  try {
    res = await put('/fn/register', { url: url.href });
  } catch (err: any) {
    throw new Error(
      `${err.message}; make sure the URL is where your Inngest server API endpoints is located`
    );
  }

  const json = await res.json();
  if (!res.ok) {
    try {
      const body = failedResponseBody.parse(json);
      let errorCode: RegistrationErrorCode | undefined = undefined;

      if (isRegistrationErrorCode(body.data.error_code)) {
        errorCode = body.data.error_code;
      }

      return {
        errorCode,
        headers: body.data.response_headers ?? {},
        statusCode: body.data.response_status_code,
      };
    } catch {
      // We don't want a Zod error to break the whole flow. Instead, we'll
      // return a minimal registration failure object.
      return {
        errorCode: undefined,
        headers: {},
        statusCode: undefined,
      };
    }
  }
}
