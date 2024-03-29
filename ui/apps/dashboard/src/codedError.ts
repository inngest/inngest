import { z } from 'zod';

const codes = [
  'account_mismatch',
  'app_mismatch',
  'app_uninitialized',
  'batch_size_too_large',
  'env_archived',
  'env_mismatch',
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
  'invalid_function',
  'invalid_signing_key',
  'missing_branch_env_name',
  'missing_signing_key',
  'no_functions',
  'not_sdk',
  'too_many_pings',
  'unknown',
  'url_invalid',

  // Deprecated
  'forbidden',
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
