import { useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import { useOrganization } from '@clerk/tanstack-react-start';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { trackKeyCreated } from '@/utils/analyticsEvents';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import { GrantPicker } from './GrantPicker';
import { APIKeyGrantsQuery, defaultSelection, permittedGrants } from './grants';
import {
  EnvironmentSelect,
  useEnvironmentSelection,
} from './EnvironmentSelect';
import { apiKeyErrorMessage } from './errorMessage';
import { RevealKeyCard } from './RevealKeyCard';
import { validateAPIKeyName } from './validation';

const Mutation = graphql(`
  mutation CreateAPIKey($input: CreateAPIKeyInput!) {
    createAPIKey(input: $input) {
      plaintextKey
      apiKey {
        id
        name
        createdAt
        maskedKey
        env {
          id
          name
        }
      }
    }
  }
`);

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function CreateAPIKeyModal({ isOpen, onClose }: Props) {
  const [name, setName] = useState('');
  const [plaintextKey, setPlaintextKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Tracks whether the user closed the modal while a mutation was in flight,
  // so we can drop the plaintext response on the floor instead of stashing it
  // in state (which would leak the key into the next modal-open).
  const cancelledRef = useRef(false);

  const { selectedEnv, setSelectedEnv, envGroups } =
    useEnvironmentSelection(isOpen);

  const { membership } = useOrganization();
  const isAdmin = membership?.role === 'org:admin';

  const grantsRes = useGraphQLQuery({
    query: APIKeyGrantsQuery,
    variables: {},
  });
  const catalog = grantsRes.data?.apiKeyGrants ?? [];
  const policy = grantsRes.data?.account.memberAPIKeyPolicy;
  // Admins may select anything in the catalog; members are narrowed to the
  // account policy. The server enforces this too — this only avoids offering a
  // toggle that would be rejected.
  const permitted = permittedGrants(catalog, policy, isAdmin);
  const [selectedGrants, setSelectedGrants] = useState<string[] | null>(null);
  // Read Only by default, narrowed to what the caller may mint.
  const grants =
    selectedGrants ??
    (catalog.length > 0 ? defaultSelection(catalog, permitted) : []);
  const [, create] = useMutation(Mutation);

  async function submit() {
    setError(null);
    const nameErr = validateAPIKeyName(name);
    if (nameErr) {
      setError(nameErr);
      return;
    }
    if (!selectedEnv) {
      setError('Select an environment.');
      return;
    }
    if (grants.length === 0) {
      setError('Select at least one permission.');
      return;
    }
    const trimmed = name.trim();

    cancelledRef.current = false;
    setIsSubmitting(true);
    try {
      const res = await create(
        { input: { name: trimmed, workspaceID: selectedEnv.id, grants } },
        { additionalTypenames: ['APIKey'] },
      );
      if (cancelledRef.current) {
        return;
      }
      if (res.error) {
        setError(apiKeyErrorMessage(res.error, 'Could not create API key.'));
        return;
      }
      const pt = res.data?.createAPIKey?.plaintextKey;
      if (!pt) {
        setError('Unexpected response from server.');
        return;
      }
      setPlaintextKey(pt);
      trackKeyCreated({
        feature: 'api-keys',
        keyID: res.data.createAPIKey.apiKey.id,
        envID: selectedEnv.id,
        grantCount: grants.length,
        surface: 'dashboard',
      });
    } finally {
      if (!cancelledRef.current) {
        setIsSubmitting(false);
      }
    }
  }

  function close() {
    cancelledRef.current = true;
    setName('');
    setSelectedEnv(null);
    setSelectedGrants(null);
    setPlaintextKey(null);
    setError(null);
    setIsSubmitting(false);
    onClose();
  }

  const inRevealStep = plaintextKey !== null;

  return (
    <Modal
      className="w-full max-w-xl overflow-visible"
      isOpen={isOpen}
      onClose={close}
    >
      <Modal.Header>
        {inRevealStep ? 'Copy your API key' : 'Create API key'}
      </Modal.Header>

      <Modal.Body>
        {inRevealStep ? (
          <RevealKeyCard plaintextKey={plaintextKey} />
        ) : (
          <div className="flex flex-col gap-5">
            <p className="text-subtle text-sm">
              Generate an API key to give your applications secure access to
              Inngest. You can remove keys at any time.
            </p>

            <div className="flex flex-col gap-2">
              <label
                htmlFor="api-key-name"
                className="text-basis text-sm font-medium"
              >
                API key name
              </label>
              <Input
                id="api-key-name"
                placeholder="eg. my-api-key"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isSubmitting}
              />
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-basis text-sm font-medium">
                Environment
              </label>
              <EnvironmentSelect
                groups={envGroups}
                value={selectedEnv}
                onChange={setSelectedEnv}
              />
            </div>

            {catalog.length > 0 && (
              <GrantPicker
                grants={catalog}
                selected={grants}
                onChange={setSelectedGrants}
                permitted={permitted}
                disabled={isSubmitting}
                restrictionNote={
                  isAdmin
                    ? undefined
                    : 'Some permissions are unavailable on this account. Ask an org admin to change what members may grant.'
                }
              />
            )}

            {error && <Alert severity="error">{error}</Alert>}
          </div>
        )}
      </Modal.Body>

      <Modal.Footer>
        <div className="flex justify-end gap-2">
          {inRevealStep ? (
            <Button kind="primary" label="Done" onClick={close} />
          ) : (
            <>
              <Button
                appearance="outlined"
                kind="secondary"
                label="Cancel"
                onClick={close}
                disabled={isSubmitting}
              />
              <Button
                kind="primary"
                label="Generate key"
                onClick={submit}
                loading={isSubmitting}
                disabled={isSubmitting}
              />
            </>
          )}
        </div>
      </Modal.Footer>
    </Modal>
  );
}
