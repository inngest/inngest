import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createServer } from 'vite';
import { describe, expect, it } from 'vitest';

describe('standalone sandbox verification preview', () => {
  it('is excluded from normal development scripts', () => {
    const { scripts } = JSON.parse(
      readFileSync(new URL('./package.json', import.meta.url), 'utf8'),
    ) as {
      scripts: Record<string, string>;
    };
    const previewCommands = Object.entries(scripts).filter(([, command]) =>
      command.includes('vite.sandbox-verification.config.ts'),
    );
    expect(previewCommands).toHaveLength(1);
    expect(scripts.dev).toContain('pnpm:dev:*');
    expect(previewCommands[0][0].startsWith('dev:')).toBe(false);
  });

  it('resolves the empty-state analytics import to the local stub through Vite', async () => {
    const cacheDir = mkdtempSync(join(tmpdir(), 'sandbox-verification-test-'));
    const server = await createServer({
      cacheDir,
      configFile: fileURLToPath(
        new URL('./vite.sandbox-verification.config.ts', import.meta.url),
      ),
      envFile: false,
      logLevel: 'silent',
      server: { middlewareMode: true, open: false, watch: null, ws: false },
      optimizeDeps: { noDiscovery: true, include: [] },
    });
    try {
      const importer = fileURLToPath(
        new URL(
          './src/components/Sandboxes/SandboxesEmptyState.tsx',
          import.meta.url,
        ),
      );
      const resolved =
        await server.environments.client.pluginContainer.resolveId(
          '@/utils/analyticsEvents',
          importer,
        );
      expect(resolved?.id).toBe(
        fileURLToPath(
          new URL(
            './src/components/Sandboxes/verification-demo/analytics.ts',
            import.meta.url,
          ),
        ),
      );

      const transformed = await server.transformRequest(
        '/src/components/Sandboxes/SandboxesEmptyState.tsx',
      );
      expect(transformed?.code).toContain(
        '/src/components/Sandboxes/verification-demo/analytics.ts',
      );
      expect(transformed?.code).not.toContain('/src/utils/analyticsEvents.ts');
    } finally {
      await server.close();
      rmSync(cacheDir, { recursive: true, force: true });
    }
  });
});
