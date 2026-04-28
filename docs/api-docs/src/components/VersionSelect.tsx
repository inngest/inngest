import { useRouter } from '@tanstack/react-router';

export function VersionSelect() {
  const router = useRouter();
  const pathname = router.state.location.pathname;

  const currentVersion = pathname.startsWith('/docs/v1') ? 'v1' : 'v2';

  function handleChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const next = e.target.value;
    // Swap the version prefix, falling back to the version index page
    const newPath = pathname.replace(/^\/docs\/(v1|v2)/, `/docs/${next}`);
    router.navigate({ to: newPath === pathname ? `/docs/${next}` : newPath });
  }

  return (
    <select
      value={currentVersion}
      onChange={handleChange}
      className="border-fd-border bg-fd-background text-fd-foreground rounded border px-2 py-1 text-sm"
    >
      <option value="v1">API v1</option>
      <option value="v2">API v2</option>
    </select>
  );
}
