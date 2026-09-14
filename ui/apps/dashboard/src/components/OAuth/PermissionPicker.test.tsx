import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import { PermissionPicker, type PermissionLevel } from './PermissionPicker';

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
