import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import { CredentialForm } from './CredentialForm';

const props = {
  name: '',
  nameLabel: 'Key name (required)',
  nameRequired: true,
  onNameChange: () => {},
  allEnvironments: false,
  onAllEnvironmentsChange: () => {},
  environment: null,
  environmentGroups: [],
  onEnvironmentChange: () => {},
  permissions: null,
  selectedResourceCount: 0,
  actions: null,
  disabled: false,
};

describe('credential form validation', () => {
  it('marks the name as required without showing errors before submission', () => {
    const html = renderToStaticMarkup(<CredentialForm {...props} />);
    expect(html).toContain('for="credential-name"');
    expect(html.match(/<input[^>]*id="credential-name"[^>]*>/)?.[0]).toContain(
      'required=""',
    );
    expect(html).not.toContain('aria-invalid="true"');
    expect(html).not.toContain('border-error');
  });

  it('shows each missing field with the shared error styles', () => {
    const html = renderToStaticMarkup(
      <CredentialForm
        {...props}
        fieldErrors={{
          name: 'Name is required.',
          environment: 'Select an environment.',
          permissions: 'Select at least one permission.',
        }}
      />,
    );
    expect(html.match(/aria-invalid="true"/g)).toHaveLength(3);
    expect(html).toContain('border-error');
    expect(html).toContain('text-error text-sm');
    expect(html).toContain('Name is required.');
    expect(html).toContain('Select an environment.');
    expect(html).toContain('Select at least one permission.');
    expect(html).toContain('aria-describedby="credential-environment-error"');
    expect(html).toContain('aria-describedby="credential-permissions-error"');
  });

  it('does not make OAuth session names required', () => {
    const html = renderToStaticMarkup(
      <CredentialForm
        {...props}
        nameLabel="Session name"
        nameRequired={false}
      />,
    );
    expect(
      html.match(/<input[^>]*id="credential-name"[^>]*>/)?.[0],
    ).not.toContain('required=""');
  });
});
