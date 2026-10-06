/**
 * @module
 * Reads a span's origin: the library that created a step or span group on the
 * user's behalf, as `<package>@<version>`, such as `@inngest/ci@0.1.0`.
 */

/**
 * Whether Inngest's own libraries created this row (`inngest` or any
 * `@inngest/*` package), rather than the user's code.
 */
export function isInngestOrigin(origin: string | null | undefined): boolean {
  if (!origin) {
    return false;
  }

  // The package is the origin without a trailing `@<version>`; a scoped
  // package starts with `@`, so an `@` at 0 is not a version.
  const at = origin.lastIndexOf('@');
  const pkg = at > 0 ? origin.slice(0, at) : origin;

  return pkg === 'inngest' || pkg.startsWith('@inngest/');
}

/**
 * Whether to draw a row or segment faded: Inngest added it, and it didn't fail.
 * A failure is never drawn quieter than the user's own work.
 */
export function isDimmed(origin: string | null | undefined, status: string | undefined): boolean {
  return isInngestOrigin(origin) && status !== 'FAILED';
}
