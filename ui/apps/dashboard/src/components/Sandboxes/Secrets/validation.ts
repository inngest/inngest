export type SecretInput = { name: string; value: string };

export const maxSecretImportCount = 256;
const encoder = new TextEncoder();

export function validateSecretName(name: string): string | undefined {
  if (!name) return 'Enter a name.';
  if (encoder.encode(name).length > 256)
    return 'Use at most 256 bytes for the name.';
  if (
    /^\p{White_Space}|\p{White_Space}$/u.test(name) ||
    /[=\r\n\0]/u.test(name)
  ) {
    return 'Names cannot contain =, line breaks, NUL, or surrounding whitespace.';
  }
}

export function validateSecretValue(value: string): string | undefined {
  if (value.includes('\0')) return 'Values cannot contain NUL.';
  if (encoder.encode(value).length > 64 * 1024)
    return 'Use at most 64 KiB for a value.';
}

export function validateSecretInputs(
  inputs: SecretInput[],
  existingNames: string[],
) {
  const existing = new Set(existingNames);
  const counts = new Map<string, number>();
  for (const { name } of inputs) counts.set(name, (counts.get(name) ?? 0) + 1);
  return inputs.map(({ name, value }) => ({
    name:
      validateSecretName(name) ??
      ((counts.get(name) ?? 0) > 1
        ? 'This name appears more than once.'
        : existing.has(name)
          ? 'Already saved. Replace its value from the secret list.'
          : undefined),
    value: validateSecretValue(value),
  }));
}
