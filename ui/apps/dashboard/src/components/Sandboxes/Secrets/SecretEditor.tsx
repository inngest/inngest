import { useEffect, useRef, useState, type ClipboardEvent } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { RiAddLine, RiArrowLeftLine, RiCloseLine } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { secretErrorMessage } from './errorMessage';
import { parseEnv } from './parseEnv';
import {
  CreateSandboxSecretDocument,
  ReplaceSandboxSecretDocument,
  secretQueryContext,
  type SandboxSecret,
} from './queries';
import { SecretValueInput } from './SecretValueInput';
import {
  maxSecretImportCount,
  validateSecretInputs,
  validateSecretValue,
  type SecretInput,
} from './validation';

type Draft = SecretInput & { id: number };
type Props = {
  environmentID: string;
  existingSecrets: SandboxSecret[];
  replacing?: SandboxSecret;
  onBack: () => void;
  onSaved: () => void;
  onDirtyChange: (dirty: boolean) => void;
  onSavingChange: (saving: boolean) => void;
};

export function SecretEditor({
  environmentID,
  existingSecrets,
  replacing,
  onBack,
  onSaved,
  onDirtyChange,
  onSavingChange,
}: Props) {
  const nextID = useRef(1);
  const cancelled = useRef(false);
  const submitting = useRef(false);
  const [rows, setRows] = useState<Draft[]>([
    { id: 0, name: replacing?.name ?? '', value: '' },
  ]);
  const [error, setError] = useState<string>();
  const [pasteMessage, setPasteMessage] = useState<string>();
  const [attempted, setAttempted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [, create] = useMutation(CreateSandboxSecretDocument);
  const [, replace] = useMutation(ReplaceSandboxSecretDocument);
  const dirty = rows.some(
    (row) => row.value !== '' || (!replacing && row.name !== ''),
  );
  const selected = rows.filter(
    (row) => replacing || row.name !== '' || row.value !== '',
  );
  const errors = validateSecretInputs(
    rows,
    replacing ? [] : existingSecrets.map((secret) => secret.name),
  );

  useEffect(() => {
    cancelled.current = false;
    return () => {
      cancelled.current = true;
    };
  }, []);
  useEffect(() => {
    onDirtyChange(dirty);
  }, [dirty, onDirtyChange]);

  function update(id: number, changes: Partial<SecretInput>) {
    setRows((current) =>
      current.map((row) => (row.id === id ? { ...row, ...changes } : row)),
    );
    setError(undefined);
  }

  function paste(event: ClipboardEvent<HTMLInputElement>, id: number) {
    const text = event.clipboardData.getData('text/plain');
    if (!/[=\r\n]/.test(text)) return;
    event.preventDefault();
    const parsed = parseEnv(text);
    if (!parsed.ok) {
      setError(parsed.error);
      return;
    }
    const row = rows.find((entry) => entry.id === id);
    const keep = rows.filter(
      (entry) => entry.id !== id || Boolean(row?.name || row?.value),
    );
    if (keep.length + parsed.entries.length > maxSecretImportCount) {
      setError(`Add up to ${maxSecretImportCount} secrets at a time.`);
      return;
    }
    setRows([
      ...keep,
      ...parsed.entries.map((entry) => ({ ...entry, id: nextID.current++ })),
    ]);
    setAttempted(true);
    setError(undefined);
    setPasteMessage(
      `${parsed.entries.length} ${parsed.entries.length === 1 ? 'secret' : 'secrets'} imported. Review before saving.`,
    );
  }

  async function submit() {
    if (submitting.current) return;
    setAttempted(true);
    setError(undefined);
    const selectedErrors = validateSecretInputs(
      selected,
      replacing ? [] : existingSecrets.map((secret) => secret.name),
    );
    if (
      !selected.length ||
      selectedErrors.some((item) => item.name || item.value)
    )
      return;
    submitting.current = true;
    setSaving(true);
    onSavingChange(true);
    let saved = 0;
    try {
      for (const row of selected) {
        if (cancelled.current) return;
        const response = replacing
          ? await replace(
              {
                input: {
                  workspaceID: environmentID,
                  id: replacing.id,
                  value: row.value,
                },
              },
              secretQueryContext,
            )
          : await create(
              {
                input: {
                  workspaceID: environmentID,
                  name: row.name,
                  value: row.value,
                },
              },
              secretQueryContext,
            );
        if (cancelled.current) return;
        if (response.error || !response.data) {
          const fallback = saved
            ? `${saved} saved. Could not confirm the next save. Review the list before trying again.`
            : 'Could not save the secret. Review the list before trying again.';
          setError(
            response.error
              ? secretErrorMessage(response.error, fallback)
              : fallback,
          );
          return;
        }
        saved++;
        // Clear successful values immediately; only unsaved rows can be resubmitted.
        setRows((current) => current.filter((entry) => entry.id !== row.id));
      }
      toast.success(
        replacing
          ? 'Secret value replaced'
          : `${saved} ${saved === 1 ? 'secret' : 'secrets'} saved`,
      );
      onDirtyChange(false);
      onSaved();
    } catch {
      if (!cancelled.current)
        setError(
          'Could not confirm the save. Review the secret list before trying again.',
        );
    } finally {
      submitting.current = false;
      if (!cancelled.current) {
        setSaving(false);
        onSavingChange(false);
      }
    }
  }

  return (
    <form
      className="flex flex-col gap-5 p-4"
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <div>
        <Button
          kind="secondary"
          appearance="ghost"
          size="small"
          icon={<RiArrowLeftLine />}
          iconSide="left"
          label="All Secrets"
          disabled={saving}
          onClick={onBack}
          className="-ml-2 mb-3"
        />
        <h2 className="text-basis text-base font-medium">
          {replacing ? 'Replace Value' : 'Add Secrets'}
        </h2>
        <p className="text-muted mt-1 text-sm">
          {replacing
            ? 'The new value is used by future launches. Running sandboxes and snapshots keep values they already received.'
            : 'Paste .env contents into a name field to fill in multiple secrets, or add them one at a time.'}
        </p>
      </div>

      {error && (
        <div role="alert">
          <Alert severity="error">{error}</Alert>
        </div>
      )}
      {pasteMessage && (
        <p role="status" className="text-muted text-xs">
          {pasteMessage}
        </p>
      )}

      <div className="flex flex-col gap-3">
        {rows.map((row, index) => {
          const rowErrors = errors[index];
          const showError =
            attempted && (Boolean(row.name || row.value) || rows.length === 1);
          return (
            <div
              key={row.id}
              className="border-subtle flex flex-col gap-3 rounded-md border p-3"
            >
              <div className="flex items-end gap-2">
                <div className="min-w-0 flex-1">
                  <label
                    htmlFor={`secret-name-${row.id}`}
                    className="text-muted mb-1.5 block text-xs"
                  >
                    Name
                  </label>
                  {replacing ? (
                    <p className="text-basis break-all font-mono text-sm">
                      {replacing.name}
                    </p>
                  ) : (
                    <Input
                      id={`secret-name-${row.id}`}
                      aria-label={`Secret ${index + 1} name`}
                      placeholder="OPENAI_API_KEY"
                      value={row.name}
                      onChange={(event) =>
                        update(row.id, { name: event.target.value })
                      }
                      onPaste={(event) => paste(event, row.id)}
                      disabled={saving}
                      spellCheck={false}
                      className="font-mono text-xs"
                      error={showError ? rowErrors?.name : undefined}
                    />
                  )}
                </div>
                {!replacing && rows.length > 1 && (
                  <Button
                    kind="secondary"
                    appearance="ghost"
                    size="small"
                    icon={<RiCloseLine />}
                    aria-label={`Remove secret ${index + 1}`}
                    disabled={saving}
                    onClick={() =>
                      setRows((current) =>
                        current.filter((entry) => entry.id !== row.id),
                      )
                    }
                  />
                )}
              </div>
              <div>
                <label
                  htmlFor={`secret-value-${row.id}`}
                  className="text-muted mb-1.5 block text-xs"
                >
                  {replacing ? 'New Value' : 'Value'}
                </label>
                <SecretValueInput
                  id={`secret-value-${row.id}`}
                  label={`Secret ${index + 1} value`}
                  value={row.value}
                  onChange={(value) => update(row.id, { value })}
                  disabled={saving}
                  error={showError ? validateSecretValue(row.value) : undefined}
                />
              </div>
            </div>
          );
        })}
        {!replacing && (
          <Button
            kind="secondary"
            appearance="ghost"
            size="small"
            icon={<RiAddLine />}
            iconSide="left"
            label="Add Another"
            className="self-start"
            disabled={saving || rows.length >= maxSecretImportCount}
            onClick={() =>
              setRows((current) => [
                ...current,
                { id: nextID.current++, name: '', value: '' },
              ])
            }
          />
        )}
      </div>

      <p className="text-muted text-xs">
        Values are encrypted and cannot be viewed after saving. Each name
        becomes its sandbox environment variable name.
      </p>
      <div className="flex justify-end gap-2">
        <Button
          kind="secondary"
          appearance="outlined"
          label="Cancel"
          disabled={saving}
          onClick={onBack}
        />
        <Button
          type="submit"
          label={
            replacing
              ? 'Replace Value'
              : `Save ${selected.length > 1 ? `${selected.length} Secrets` : 'Secret'}`
          }
          loading={saving}
          disabled={saving || (!replacing && selected.length === 0)}
        />
      </div>
    </form>
  );
}
