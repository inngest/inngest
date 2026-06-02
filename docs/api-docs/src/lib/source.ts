import { docs } from 'collections/server';
import { loader } from 'fumadocs-core/source';

export const source = loader({
  baseUrl: '/',
  source: docs.toFumadocsSource(),
});

/**
 * Strip the trailing `.md` from the last segment of a request path so it maps
 * back to the same slugs `source.getPage()` uses. `index.md` at the root
 * resolves to the empty-slug homepage.
 */
export function markdownPathToSlugs(segs: string[]): string[] {
  if (segs.length === 0) return [];
  const out = [...segs];
  out[out.length - 1] = out[out.length - 1].replace(/\.md$/, '');
  if (out.length === 1 && out[0] === 'index') out.pop();
  return out;
}
