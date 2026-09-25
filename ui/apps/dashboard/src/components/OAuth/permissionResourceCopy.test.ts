import { expect, it } from 'vitest';

import { permissionResourceCopy } from './permissionResourceCopy';

it.each([
  ['api_keys', 'API keys'],
  ['event_keys', 'Event keys'],
  ['new__resource', 'New Resource'],
])('labels %s without a description', (resource, label) => {
  expect(permissionResourceCopy(resource)).toEqual({
    label,
    description: null,
  });
});

it('keeps the webhook credential warning', () => {
  expect(permissionResourceCopy('webhooks').description).toBe(
    'Read access includes URLs that can send events. Copied URLs still work after key or session revocation. Revoke the webhook to disable them.',
  );
});
