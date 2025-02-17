import { z } from 'zod';

// This is a TypeScript implementation of the syscode package in Inngest OSS

const codes = [
  'account_mismatch',
  'app_mismatch',
  'app_uninitialized',
  'batch_size_too_large',
  'env_archived',
  'env_mismatch',
  'env_unspecified',
  'env_unspecified',
  'host_private',
  'http_bad_request',
  'http_forbidden',
  'http_internal_server_error',
  'http_method_not_allowed',
  'http_not_found',
  'http_unauthorized',
  'http_unreachable',
  'http_unsupported_protocol',
  'no_functions',
  'not_sdk',
  'response_not_signed',
  'server_kind_mismatch',
  'sig_verification_failed',
  'signing_key_invalid',
  'signing_key_unspecified',
  'too_many_pings',
  'unknown',
  'url_invalid',

  // Deprecated
  'invalid_function',
  'missing_signing_key',
  'invalid_signing_key',
  'forbidden',
  'missing_branch_env_name',
  'internal_server_error',
  'unauthorized',
  'unreachable',
  'unsupported_protocol',
  'url_not_found',
] as const;
export type ErrorCode = (typeof codes)[number];
export function isErrorCode(value: unknown): value is ErrorCode {
  return codes.includes(value as ErrorCode);
}

const codedErrorSchema = z.object({
  code: z.string(),
  data: z.unknown().optional(),
  message: z.string().optional(),
});

export type CodedError = z.infer<typeof codedErrorSchema>;

export const httpDataSchema = z.object({
  headers: z.record(z.array(z.string())),
  statusCode: z.number(),
});

export const dataMultiErr = z.object({
  errors: z.array(codedErrorSchema),
});

export function parseErrorData(
  data: unknown
): z.infer<typeof httpDataSchema> | z.infer<typeof dataMultiErr> | null {
  if (data === null || data === undefined) {
    return null;
  }

  if (typeof data !== 'string') {
    console.error('error data is not a string:', data);
    return null;
  }

  let obj: unknown;
  try {
    obj = JSON.parse(data);
  } catch {
    console.error('failed to parse error data:', data);
    return null;
  }

  const httpData = httpDataSchema.safeParse(obj);
  if (httpData.success) {
    return httpData.data;
  }

  const multiErr = dataMultiErr.safeParse(obj);
  if (multiErr.success) {
    return multiErr.data;
  }

  console.error('unknown error data format:', data);
  return null;
}
