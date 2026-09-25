import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import { APIKeyPermissions } from './APIKeyPermissions';

describe('API key permissions', () => {
  it('groups grants by resource and shows readable access levels', () => {
    const html = renderToStaticMarkup(
      <APIKeyPermissions
        permissions={['apps:read:*', 'apps:write:*', 'accounts:read:*']}
      />,
    );

    expect(html).toContain('Resource');
    expect(html).toContain('Access');
    expect(html.match(/>Apps</g)).toHaveLength(1);
    expect(html.indexOf('>Accounts<')).toBeLessThan(html.indexOf('>Apps<'));
    expect(html).toContain('>Read<');
    expect(html).toContain('>Write<');
    expect(html).not.toContain('apps:read:*');
  });

  it('keeps operation-specific grants explicit', () => {
    const html = renderToStaticMarkup(
      <APIKeyPermissions permissions={['apps:read:get', 'apps:write:sync']} />,
    );

    expect(html).toContain('Read: get');
    expect(html).toContain('Write: sync');
  });
});
