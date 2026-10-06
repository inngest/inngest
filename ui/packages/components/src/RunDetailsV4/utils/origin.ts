/**
 * @module
 * Reads a span's origin: the library that created a step or span group on the
 * user's behalf, as `<package>@<version>`, such as `@inngest/ci@0.1.0`.
 */

/** The package part of an origin, without a trailing `@<version>` */
function originPackage(origin: string): string {
  const at = origin.lastIndexOf('@');

  // A scoped package starts with `@`, so an `@` at 0 is not a version.
  return at > 0 ? origin.slice(0, at) : origin;
}

/**
 * Whether Inngest's own libraries created this row (`inngest` or any
 * `@inngest/*` package), rather than the user's code.
 */
export function isInngestOrigin(origin: string | null | undefined): boolean {
  if (!origin) {
    return false;
  }

  const pkg = originPackage(origin);

  return pkg === 'inngest' || pkg.startsWith('@inngest/');
}
