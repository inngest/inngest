/**
 * Server-side OpenAPI instance for v2.
 *
 * Uses /api-specs/v2.json (a public static asset) as the logical key so that:
 *  - The API playground can POST to the correct base URL.
 *  - The generated MDX's document prop matches this key.
 */
import { createOpenAPI } from 'fumadocs-openapi/server';

export const openapi = createOpenAPI({
  input: ['/api-specs/v2.json'],
});
