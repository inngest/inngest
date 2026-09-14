import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import { PermissionPicker, type PermissionLevel } from './PermissionPicker';

describe('permission controls', () => {
  it('omits unavailable access and defaults to none', () => {
    const html = renderToStaticMarkup(
      <PermissionPicker
        groups={[{ resource: 'events', read: [], write: ['events:write:*'] }]}
        levels={{}}
        onChange={() => {}}
      />,
    );

    expect(html).not.toContain('aria-label="Events: Read"');
    expect(html).toContain('aria-label="Events: None"');
    expect(
      html.match(/<input[^>]*aria-label="Events: None"[^>]*>/)?.[0],
    ).toContain('checked=""');
    expect(html).toContain('aria-label="Events: Write"');
  });

  it('disables both shortcuts and radios while submitting', () => {
    const html = renderToStaticMarkup(
      <PermissionPicker
        groups={[
          { resource: 'apps', read: ['apps:read:*'], write: ['apps:write:*'] },
        ]}
        levels={{ apps: 'read' }}
        disabled
        onChange={() => {}}
      />,
    );

    expect(html.match(/<input[^>]+disabled=""/g)).toHaveLength(3);
    expect(html.match(/<button[^>]+disabled=""/g)).toHaveLength(3);
  });
});

describe('webhook consent', () => {
  it.each<PermissionLevel>(['read', 'write'])(
    'discloses durable event access for %s',
    (level) => {
      const html = renderToStaticMarkup(
        <PermissionPicker
          groups={[
            {
              resource: 'webhooks',
              read: ['webhooks:read:*'],
              write: ['webhooks:write:*'],
            },
          ]}
          levels={{ webhooks: level }}
          onChange={() => {}}
        />,
      );

      expect(html).toContain('URLs that can send events');
      expect(html).toContain('still work after logout or session revocation');
      expect(html).toContain('Revoke the webhook to disable them');
      expect(html).not.toContain('text-subtle truncate');
    },
  );
});
