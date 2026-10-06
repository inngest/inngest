import { useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useInngestAPIFetch } from '@/queries/useInngestAPIFetch';

import { buildLabel, imageApi, imageBytes, type ImageTag } from './imageApi';

export function ImagesPanel() {
  const env = useEnvironment();
  const fetch = useInngestAPIFetch(env.slug);
  const api = useMemo(() => imageApi(fetch), [fetch]);
  const client = useQueryClient();
  const key = ['sandbox-images', env.id];
  const [selected, setSelected] = useState<string>();
  const [selectedBuild, setSelectedBuild] = useState<string>();
  const [deleteTarget, setDeleteTarget] = useState<ImageTag>();
  const images = useInfiniteQuery({
    queryKey: [...key, 'catalog'],
    initialPageParam: '',
    queryFn: ({ pageParam }) => api.list(pageParam),
    getNextPageParam: (last) => last.nextCursor || undefined,
    refetchInterval: 10_000,
  });
  const usage = useQuery({
    queryKey: [...key, 'usage'],
    queryFn: api.usage,
    refetchInterval: 10_000,
  });
  const builds = useInfiniteQuery({
    queryKey: [...key, 'builds'],
    initialPageParam: '',
    queryFn: ({ pageParam }) => api.builds(pageParam),
    getNextPageParam: (last) => last.nextCursor || undefined,
    refetchInterval: 5000,
  });
  const detail = useInfiniteQuery({
    queryKey: [...key, 'image', selected],
    enabled: !!selected,
    initialPageParam: '',
    queryFn: ({ pageParam }) => api.get(selected!, pageParam),
    getNextPageParam: (last) => last.data.nextArtifactCursor || undefined,
    refetchInterval: 10_000,
  });
  const build = useQuery({
    queryKey: [...key, 'build', selectedBuild],
    enabled: !!selectedBuild,
    queryFn: () => api.build(selectedBuild!),
    refetchInterval: (query) =>
      ['ready', 'failed'].includes(query.state.data?.data.status ?? '')
        ? false
        : 2000,
  });
  const remove = useMutation({
    mutationFn: ({ name, tag }: { name: string; tag: ImageTag }) =>
      api.deleteTag(name, tag),
    onSuccess: () => {
      setDeleteTarget(undefined);
      return client.invalidateQueries({ queryKey: key });
    },
  });
  const cancel = useMutation({
    mutationFn: api.cancel,
    onSuccess: () => client.invalidateQueries({ queryKey: key }),
  });
  const error =
    images.error ??
    usage.error ??
    builds.error ??
    detail.error ??
    build.error ??
    remove.error ??
    cancel.error;
  const image = detail.data?.pages[0]?.data;
  const activeBuild = build.data?.data;
  const rows = images.data?.pages.flatMap((p) => p.data) ?? [];
  const buildRows = builds.data?.pages.flatMap((p) => p.data) ?? [];
  const refresh = () => {
    remove.reset();
    cancel.reset();
    void client.invalidateQueries({ queryKey: key });
  };

  return (
    <div className="bg-canvasBase text-basis h-full overflow-auto p-6">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-lg font-medium">Images</h2>
          <p className="text-muted text-sm">
            Custom images are private to this environment. Inngest images are
            available to everyone.
          </p>
        </div>
        <Button
          kind="secondary"
          appearance="outlined"
          label="Refresh"
          onClick={refresh}
        />
      </div>
      {error && (
        <p role="alert" className="text-error mb-4">
          {error.message}
        </p>
      )}
      {usage.data && (
        <dl className="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
          {[
            ['Stored images', imageBytes(usage.data.data.storedBytes)],
            ['Temporary uploads', imageBytes(usage.data.data.uploadBytes)],
            [
              'Build minutes',
              (
                (usage.data.data.buildMilliseconds +
                  usage.data.data.conversionMilliseconds) /
                60_000
              ).toFixed(2),
            ],
            ['Active builds', usage.data.data.activeBuilds],
          ].map(([label, value]) => (
            <div key={label} className="border-subtle rounded border p-3">
              <dt className="text-muted text-sm">{label}</dt>
              <dd className="text-xl">{value}</dd>
            </div>
          ))}
        </dl>
      )}
      {images.isPending && <p role="status">Loading images…</p>}
      {!images.isPending && !images.error && rows.length === 0 && (
        <p>
          No images yet. Publish one with{' '}
          <code>inngest-cli image push -t app:latest .</code>
        </p>
      )}
      <table className="mb-4 w-full text-left text-sm">
        <caption className="sr-only">Image catalog</caption>
        <thead>
          <tr>
            <th className="p-2">Image</th>
            <th>Visibility</th>
            <th>Tags</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((item) => (
            <tr key={item.id} className="border-subtle border-t">
              <td className="p-2">
                <Button
                  kind="secondary"
                  appearance="ghost"
                  label={item.name}
                  onClick={() => {
                    setSelected(item.name);
                    setDeleteTarget(undefined);
                    remove.reset();
                  }}
                />
              </td>
              <td>{item.scope === 'public' ? 'Public' : 'This environment'}</td>
              <td>
                {item.tags
                  .filter((t) => !t.deleted)
                  .map((t) => t.name)
                  .join(', ') || 'No tags'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {images.hasNextPage && (
        <Button
          label="Load more images"
          kind="secondary"
          loading={images.isFetchingNextPage}
          onClick={() => void images.fetchNextPage()}
        />
      )}
      {selected && detail.isPending && (
        <p role="status">Loading image details…</p>
      )}
      {image && (
        <section
          aria-label="Image details"
          className="border-subtle my-6 rounded border p-4"
        >
          <div className="flex items-center justify-between">
            <h3 className="font-medium">{image.name}</h3>
            <Button
              label="Close image"
              appearance="ghost"
              kind="secondary"
              onClick={() => {
                setSelected(undefined);
                setDeleteTarget(undefined);
              }}
            />
          </div>
          <p className="text-muted my-2 text-sm">
            Use a tag for new sandboxes, or pin a retained digest. Existing
            sandboxes keep their original image.
          </p>
          <ul>
            {image.tags
              .filter((t) => !t.deleted)
              .map((t) => (
                <li
                  key={`${t.name}/${t.architecture}`}
                  className="flex flex-wrap items-center gap-2 py-2 text-sm"
                >
                  <code>
                    {image.name}:{t.name}
                  </code>
                  <span>
                    {t.architecture}
                    {t.immutable ? ' · Immutable' : ''}
                  </span>
                  <code className="break-all text-xs">{t.digest}</code>
                  {image.scope === 'workspace' && !t.immutable && (
                    <Button
                      label={`Delete tag ${t.name}`}
                      kind="danger"
                      appearance="ghost"
                      size="small"
                      onClick={() => {
                        remove.reset();
                        setDeleteTarget(t);
                      }}
                    />
                  )}
                </li>
              ))}
          </ul>
          {deleteTarget && (
            <div
              className="border-subtle my-3 rounded border p-3"
              role="group"
              aria-label="Confirm tag deletion"
            >
              <p className="mb-2 text-sm">
                Delete {image.name}:{deleteTarget.name}? Existing sandboxes and
                snapshots keep their pinned image. Untagged files may be removed
                after their retention period.
              </p>
              <div className="flex gap-2">
                <Button
                  kind="danger"
                  label="Confirm delete tag"
                  loading={remove.isPending}
                  onClick={() =>
                    remove.mutate({ name: image.name, tag: deleteTarget })
                  }
                />
                <Button
                  kind="secondary"
                  label="Keep tag"
                  disabled={remove.isPending}
                  onClick={() => setDeleteTarget(undefined)}
                />
              </div>
            </div>
          )}
          <h4 className="mt-4 text-sm font-medium">Retained artifacts</h4>
          {detail.data?.pages
            .flatMap((p) => p.data.artifacts)
            .map((artifact) => (
              <details
                key={artifact.digest}
                className="border-subtle border-b py-3 text-sm"
              >
                <summary className="cursor-pointer break-all">
                  <code>
                    {image.name}@sha256:{artifact.digest}
                  </code>{' '}
                  · {artifact.state} · {imageBytes(artifact.manifest.sizeBytes)}
                </summary>
                <p className="my-2">
                  {artifact.manifest.architecture} · Filesystem checksum:{' '}
                  <code className="break-all">
                    {artifact.manifest.rootfsSha256}
                  </code>
                </p>
                <pre className="bg-canvasSubtle overflow-auto rounded p-2 text-xs">
                  {JSON.stringify(artifact.manifest.config, null, 2)}
                </pre>
              </details>
            ))}
          {detail.hasNextPage && (
            <Button
              label="Load more artifacts"
              kind="secondary"
              loading={detail.isFetchingNextPage}
              onClick={() => void detail.fetchNextPage()}
            />
          )}
        </section>
      )}
      <h3 className="mb-2 mt-6 font-medium">Builds</h3>
      {builds.isPending && <p role="status">Loading builds…</p>}
      {!builds.isPending && !builds.error && buildRows.length === 0 && (
        <p className="text-muted text-sm">No builds in this environment.</p>
      )}
      <table className="mb-4 w-full text-left text-sm">
        <caption className="sr-only">Image builds</caption>
        <thead>
          <tr>
            <th className="p-2">Image</th>
            <th>Status</th>
            <th>Created</th>
            <th>Minutes</th>
          </tr>
        </thead>
        <tbody>
          {buildRows.map((item) => (
            <tr key={item.id} className="border-subtle border-t">
              <td className="p-2">
                <Button
                  kind="secondary"
                  appearance="ghost"
                  label={`${item.name}:${item.tag}`}
                  onClick={() => {
                    setSelectedBuild(item.id);
                    cancel.reset();
                  }}
                />
              </td>
              <td>{buildLabel(item)}</td>
              <td>
                <time dateTime={item.createdAt}>
                  {new Date(item.createdAt).toLocaleString()}
                </time>
              </td>
              <td>{(item.executionMilliseconds / 60_000).toFixed(2)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {builds.hasNextPage && (
        <Button
          label="Load more builds"
          kind="secondary"
          loading={builds.isFetchingNextPage}
          onClick={() => void builds.fetchNextPage()}
        />
      )}
      {activeBuild && (
        <section
          aria-label="Build details"
          className="border-subtle mt-6 rounded border p-4"
        >
          <div className="flex items-center justify-between">
            <h3 className="font-medium">
              {activeBuild.name}:{activeBuild.tag} · {buildLabel(activeBuild)}
            </h3>
            <Button
              kind="secondary"
              appearance="ghost"
              label="Close build"
              onClick={() => setSelectedBuild(undefined)}
            />
          </div>
          <p className="text-muted my-2 text-sm">
            {activeBuild.sourceType} · {activeBuild.architecture} ·{' '}
            {imageBytes(activeBuild.uploadSizeBytes)} uploaded
          </p>
          {activeBuild.errorMessage && (
            <p className="my-2 text-sm">{activeBuild.errorMessage}</p>
          )}
          {activeBuild.imageRef && (
            <p className="my-2 break-all text-sm">
              <code>{activeBuild.imageRef}</code>
            </p>
          )}
          {activeBuild.status === 'ready' && !activeBuild.tagUpdated && (
            <p className="my-2 text-sm">
              The tag changed during this build. Use the digest above to launch
              this artifact.
            </p>
          )}
          {['pending', 'uploaded'].includes(activeBuild.status) && (
            <Button
              kind="danger"
              appearance="outlined"
              label="Cancel build"
              loading={cancel.isPending}
              onClick={() => cancel.mutate(activeBuild.id)}
            />
          )}
          <pre
            aria-label="Build logs"
            className="bg-canvasSubtle mt-3 max-h-96 overflow-auto whitespace-pre-wrap rounded p-3 text-xs"
          >
            {activeBuild.logs || 'No logs yet.'}
          </pre>
        </section>
      )}
    </div>
  );
}
