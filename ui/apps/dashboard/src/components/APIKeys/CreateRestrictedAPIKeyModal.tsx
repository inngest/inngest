import { useEffect, useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import type { Option } from '@inngest/components/Select/Select';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { useEnvironments } from '@/queries/environments';
import { EnvironmentType } from '@/utils/environments';
import { CredentialForm } from '@/components/OAuth/CredentialForm';
import { credentialEnvironmentOptions } from '@/components/OAuth/credentialEnvironments';
import {
  PermissionPicker,
  type PermissionGroup,
  type PermissionLevel,
} from '@/components/OAuth/PermissionPicker';
import { selectedPermissionGrants } from '@/components/OAuth/permissionSelection';
import { apiKeyErrorMessage } from './errorMessage';
import { RevealKeyCard } from './RevealKeyCard';
import { validateRestrictedAPIKey } from './validation';

const Create = graphql(`
  mutation CreateRestrictedAPIKey($input: CreateAPICredentialInput!) {
    createAPICredential(input: $input) {
      plaintextKey
      key { id }
    }
  }
`);

const expirations: Option[] = [7, 30, 90, 365].map((days) => ({
  id: String(days),
  name: `${days} days`,
}));
const defaultExpiration: Option = { id: '30', name: '30 days' };

export function CreateRestrictedAPIKeyModal({
  groups,
  onClose,
}: {
  groups: PermissionGroup[];
  onClose: () => void;
}) {
  const [name, setName] = useState('');
  const [environment, setEnvironment] = useState<Option | null>(null);
  const [allEnvironments, setAllEnvironments] = useState(false);
  const [expiration, setExpiration] = useState(defaultExpiration);
  const [levels, setLevels] = useState<Record<string, PermissionLevel>>({});
  const [secret, setSecret] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [
    {
      data: environments,
      fetching: loadingEnvironments,
      error: environmentError,
    },
  ] = useEnvironments();
  const [{ fetching }, create] = useMutation(Create);
  const active = useRef(true);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);

  const permissions = selectedPermissionGrants(groups, levels);
  const branchEnvironment = environments?.find(
    (env) => env.type === EnvironmentType.BranchParent && !env.isArchived,
  );
  const validation = validateRestrictedAPIKey({
    name,
    allEnvironments,
    workspaceID: environment?.id,
    permissions,
  });

  async function submit() {
    if (validation || fetching) return;
    setError(null);
    const result = await create({
      input: {
        name: name.trim(),
        permissions,
        allEnvironments,
        workspaceID: allEnvironments ? null : environment?.id,
        expiresAt: new Date(
          Date.now() + Number(expiration.id) * 86_400_000,
        ).toISOString(),
      },
    });
    // discard a secret returned after closing or switching organizations
    if (!active.current) return;
    if (result.error || !result.data) {
      setError(
        result.error
          ? apiKeyErrorMessage(result.error, 'Could not create API key.')
          : 'Could not create API key.',
      );
      return;
    }
    setSecret(result.data.createAPICredential.plaintextKey);
  }

  return (
    <Modal
      isOpen
      onClose={() => {
        if (!fetching) onClose();
      }}
      className="w-full max-w-2xl overflow-visible"
    >
      <Modal.Header>
        {secret ? 'Copy your API key' : 'Create restricted v2 key'}
      </Modal.Header>
      <Modal.Body>
        {secret ? (
          <RevealKeyCard plaintextKey={secret} />
        ) : (
          <div className="flex flex-col gap-6">
            <p className="text-subtle text-sm">
              Shared keys belong to this organization and keep working if you
              leave. They work only with the v2 API. Create a new key to change
              its access.
            </p>
            <CredentialForm
              name={name}
              nameLabel="Key name"
              namePlaceholder="eg. nightly-sync"
              onNameChange={setName}
              expiration={{
                value: expiration,
                options: expirations,
                onChange: setExpiration,
              }}
              allEnvironments={allEnvironments}
              onAllEnvironmentsChange={setAllEnvironments}
              environment={environment}
              branchEnvironment={
                branchEnvironment
                  ? {
                      id: branchEnvironment.id,
                      name: 'All branch environments',
                    }
                  : undefined
              }
              onEnvironmentChange={setEnvironment}
              environmentGroups={credentialEnvironmentOptions(
                environments ?? [],
                true,
              )}
              selectedResourceCount={
                groups.filter(
                  (group) =>
                    levels[group.resource] && levels[group.resource] !== 'none',
                ).length
              }
              permissions={
                <div className="flex flex-col gap-3">
                  <p className="text-subtle text-sm">
                    Access includes future operations in each selected group.
                    Keys cannot read signing keys, event keys, or webhook
                    secrets.
                  </p>
                  <PermissionPicker
                    groups={groups}
                    levels={levels}
                    onChange={setLevels}
                    disabled={fetching}
                  />
                </div>
              }
              disabled={fetching || loadingEnvironments}
              error={
                (error || environmentError) && (
                  <Alert severity="error">
                    {error ?? 'Could not load environments.'}
                  </Alert>
                )
              }
              actions={
                <>
                  <Button
                    kind="secondary"
                    appearance="outlined"
                    label="Cancel"
                    onClick={onClose}
                    disabled={fetching}
                  />
                  <Button
                    label="Create key"
                    onClick={submit}
                    loading={fetching}
                    disabled={
                      Boolean(validation) ||
                      fetching ||
                      loadingEnvironments ||
                      Boolean(environmentError)
                    }
                  />
                </>
              }
            />
          </div>
        )}
      </Modal.Body>
      {secret && (
        <Modal.Footer>
          <Button label="Done" onClick={onClose} />
        </Modal.Footer>
      )}
    </Modal>
  );
}
