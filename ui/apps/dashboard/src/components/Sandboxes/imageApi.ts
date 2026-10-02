import { z } from 'zod';

import type { InngestAPIFetch } from '@/queries/useInngestAPIFetch';

// REST v2 uses protobuf JSON: int64 values arrive as decimal strings.
const count = z
  .union([z.number(), z.string().regex(/^\d+$/).transform(Number)])
  .pipe(z.number().int().nonnegative().safe());
const digest = z.string().regex(/^[a-f0-9]{64}$/);
const name = z.string().regex(/^(?:inngest\/)?[a-z0-9][a-z0-9._-]{0,62}$/);
const tag = z.object({
  name: z.string(),
  digest: digest.nullish(),
  generation: count,
  architecture: z.string(),
  immutable: z.boolean().default(false),
  deleted: z.boolean().default(false),
});
export const imageSchema = z.object({
  id: z.uuid(),
  name,
  scope: z.enum(['public', 'workspace']),
  createdAt: z.string(),
  tags: z.array(tag).default([]),
  artifacts: z
    .array(
      z.object({
        digest,
        state: z.string(),
        createdAt: z.string(),
        manifest: z.object({
          architecture: z.string(),
          rootfsSha256: digest,
          sizeBytes: count,
          config: z.record(z.string(), z.unknown()).default({}),
        }),
      }),
    )
    .default([]),
  nextArtifactCursor: digest.nullish(),
});
export const buildSchema = z.object({
  id: z.uuid(),
  imageId: z.uuid(),
  name,
  tag: z.string(),
  status: z.enum(['pending', 'uploaded', 'ready', 'failed']),
  state: z.enum([
    'queued',
    'running',
    'publishing',
    'succeeded',
    'failed',
    'cancelled',
  ]),
  architecture: z.string(),
  sourceType: z.string(),
  uploadSizeBytes: count,
  imageRef: z.string().default(''),
  artifactDigest: digest.nullish(),
  errorMessage: z.string().default(''),
  logs: z.string().default(''),
  tagUpdated: z.boolean().default(false),
  cacheHit: z.boolean().default(false),
  executionMilliseconds: count.default(0),
  createdAt: z.string(),
});
const usageSchema = z.object({
  storedBytes: count.default(0),
  uploadBytes: count.default(0),
  buildMilliseconds: count.default(0),
  conversionMilliseconds: count.default(0),
  activeBuilds: z.number().int().nonnegative().default(0),
});
const page = <T extends z.ZodType>(schema: T) =>
  z.object({
    data: z.array(schema).default([]),
    nextCursor: z.string().nullish(),
  });
export type CatalogImage = z.infer<typeof imageSchema>;
export type ImageBuild = z.infer<typeof buildSchema>;
export type ImageTag = z.infer<typeof tag>;

export function imageApi(fetch: InngestAPIFetch) {
  async function request<T>(
    path: string,
    schema: z.ZodType<T>,
    init?: RequestInit,
  ): Promise<T> {
    const response = await fetch(path, init);
    if (!response.ok) {
      // Avoid rendering provider errors or credentials echoed by a proxy.
      throw new Error(
        response.status === 409
          ? 'This image changed. Refresh and try again.'
          : response.status === 403
            ? 'You do not have permission to manage these images.'
            : `Image request failed (${response.status}).`,
      );
    }
    return schema.parse(await response.json());
  }
  const imagePath = (value: string) => `/v2/images/${name.parse(value)}`;
  const buildPath = (value: string) =>
    `/v2/image-builds/${z.uuid().parse(value)}`;
  return {
    list: (cursor = '') =>
      request(
        `/v2/images?limit=50&cursor=${encodeURIComponent(cursor)}`,
        page(imageSchema),
      ),
    get: (value: string, cursor = '') =>
      request(
        `${imagePath(value)}?artifactCursor=${encodeURIComponent(cursor)}`,
        z.object({ data: imageSchema }),
      ),
    usage: () => request('/v2/image-usage', z.object({ data: usageSchema })),
    builds: (cursor = '') =>
      request(
        `/v2/image-builds?limit=50&cursor=${encodeURIComponent(cursor)}`,
        page(buildSchema),
      ),
    build: (id: string) =>
      request(buildPath(id), z.object({ data: buildSchema })),
    cancel: (id: string) =>
      request(`${buildPath(id)}/cancel`, z.object({ data: buildSchema }), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: '{}',
      }),
    deleteTag: (value: string, current: ImageTag) =>
      request(
        `${imagePath(value)}/tags/${encodeURIComponent(current.name)}?expectedGeneration=${current.generation}`,
        z.object({ data: tag }),
        { method: 'DELETE' },
      ),
  };
}

export function buildLabel(build: ImageBuild): string {
  if (build.status === 'pending') return 'Awaiting upload';
  if (build.state === 'cancelled') return 'Cancelled';
  if (build.status === 'ready')
    return build.cacheHit ? 'Ready (cached)' : 'Ready';
  return build.state.charAt(0).toUpperCase() + build.state.slice(1);
}

export function imageBytes(bytes: number): string {
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KiB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MiB`;
  return `${(bytes / 1024 ** 3).toFixed(1)} GiB`;
}
