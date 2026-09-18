import { useEffect, useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import type { Option } from '@inngest/components/Select/Select';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { useEnvironments } from '@/queries/environments';
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

const expirations: Option[] = [
  ...[7, 30, 90, 365].map((days) => ({
    id: String(days),
    name: `${days} days`,
  })),
  { id: 'never', name: 'Never expires' },
];
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
  const [showValidation, setShowValidation] = useState(false);
  const form = useRef<HTMLFormElement>(null);
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
  const validation = validateRestrictedAPIKey({
    name,
    allEnvironments,
    workspaceID: environment?.id,
    permissions,
  });

  async function submit() {
    if (fetching || loadingEnvironments || environmentError) {
      return;
    }
    setError(null);
    setShowValidation(true);
    if (Object.values(validation).some(Boolean)) {
      requestAnimationFrame(() => {
        form.current
          ?.querySelector<HTMLElement>('[aria-invalid="true"]')
          ?.focus();
      });
      return;
    }
    const result = await create({
      input: {
        name: name.trim(),
        permissions,
        allEnvironments,
        workspaceID: allEnvironments ? null : environment?.id,
        expiresAt:
          expiration.id === 'never'
            ? null
            : new Date(
                Date.now() + Number(expiration.id) * 86_400_000,
              ).toISOString(),
      },
    });
    // discard a secret returned after closing or switching organizations
    if (!active.current) {
      return;
    }
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
        if (!fetching) {
          onClose();
        }
      }}
      className="w-full max-w-2xl"
    >
      <Modal.Header>
        {secret ? 'Copy your API key' : 'Create API key'}
      </Modal.Header>
      <Modal.Body>
        {secret ? (
          <RevealKeyCard plaintextKey={secret} />
        ) : (
          <form
            ref={form}
            noValidate
            onSubmit={(event) => {
              event.preventDefault();
              void submit();
            }}
            className="flex flex-col gap-6"
          >
            <p className="text-subtle text-sm">
              Keys belong to your organization and keep working if you leave.
            </p>
            <CredentialForm
              name={name}
              nameLabel="Key name (required)"
              nameRequired
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
              onEnvironmentChange={setEnvironment}
              environmentGroups={credentialEnvironmentOptions(
                environments ?? [],
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
              fieldErrors={showValidation ? validation : undefined}
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
                    type="submit"
                    loading={fetching}
                    disabled={
                      fetching ||
                      loadingEnvironments ||
                      Boolean(environmentError)
                    }
                  />
                </>
              }
            />
          </form>
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
