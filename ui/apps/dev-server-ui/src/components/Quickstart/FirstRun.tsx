import { QuietCopyButton } from './QuietCopyButton';
import { Navigate } from '@tanstack/react-router';
import { RiInformationLine } from '@remixicon/react';
import { Link } from '@inngest/components/Link';
import { useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock';
import { Select } from '@inngest/components/Select/Select';
import { parseCode } from '@inngest/components/InvokeButton/utils';
import {
  FunctionTriggerTypes,
  useGetFunctionsQuery,
  useInvokeFunctionMutation,
} from '@/store/generated';
import { useQuickstart } from './QuickstartProvider';
import { useQuickstartTracking } from './useQuickstartTracking';

export function FirstRun({
  showSetupGuide = false,
}: {
  showSetupGuide?: boolean;
}) {
  const [invokedAt, setInvokedAt] = useState<string | null>(null);
  const {
    server,
    functions,
    attempt,
    setAttempt,
    candidates,
    run,
    runs,
    completion,
  } = useQuickstart();
  const track = useQuickstartTracking(server);
  const [selectedID, setSelectedID] = useState(attempt?.functionID || '');
  const selected = functions.find(
    (fn) =>
      fn.id ===
      (selectedID || (functions.length === 1 ? functions[0]?.id : '')),
  );
  const {
    data: functionData,
    error: functionError,
    refetch: refetchFunctions,
  } = useGetFunctionsQuery(undefined, {
    pollingInterval: completion && !showSetupGuide ? 0 : 1500,
    skip: (!!completion && !showSetupGuide) || !functions.length,
  });
  const details = functionData?.functions?.find((fn) => fn.id === selected?.id);
  const acceptsPayload = details?.triggers?.some(
    (trigger) => trigger.type === FunctionTriggerTypes.Event,
  );
  const [payload, setPayload] = useState(JSON.stringify({ data: {} }, null, 2));
  const [error, setError] = useState('');
  const [invoke, invocation] = useInvokeFunctionMutation();
  const submitting = useRef(false);
  const currentAttempt = !showSetupGuide || attempt?.startedAt === invokedAt;
  const pending = currentAttempt && !!attempt?.functionID;
  const runID = showSetupGuide
    ? currentAttempt
      ? attempt?.runID || run?.id
      : undefined
    : completion?.runID || attempt?.runID || run?.id;

  async function submit() {
    if (!selected || !details || pending || submitting.current) return;
    let parsed: ReturnType<typeof parseCode>;
    try {
      parsed = acceptsPayload
        ? parseCode(payload)
        : { data: {}, user: null, meta: null };
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Invalid payload');
      return;
    }
    submitting.current = true;
    track('invoke_clicked', { path: 'manual' });
    setError('');
    const startedAt = new Date().toISOString();
    setInvokedAt(startedAt);
    setAttempt({ functionID: selected.id, startedAt });
    try {
      const response = await invoke({
        functionSlug: selected.slug,
        ...parsed,
      }).unwrap();
      if (!response.invokeFunction) {
        setAttempt(null);
        setError(
          'The invocation was not accepted. Check your app connection and try again.',
        );
      }
    } catch {
      // A lost response may still have started a run. Keep observing before retrying.
      setError(
        'Could not confirm the invocation. Check Runs before sending another invocation.',
      );
    } finally {
      submitting.current = false;
    }
  }
  if (runID) {
    return <Navigate to="/runs" search={{ runID }} replace />;
  }
  return (
    <section className="space-y-4" aria-labelledby="first-run">
      <div className="space-y-1">
        <h2 id="first-run" className="font-medium">
          {functions.length && !showSetupGuide
            ? 'Invoke your first function'
            : '3. Invoke your first function'}
        </h2>
        <p className="text-muted text-sm">
          Choose a function, invoke it, and inspect its run.
        </p>
      </div>
      {functions.length > 0 &&
        attempt?.mode === 'agent' &&
        !attempt.functionID && (
          <p className="text-muted text-sm" role="status">
            Choose the function your agent is testing so we can follow its run.
          </p>
        )}
      <Select
        label="Function"
        isLabelVisible={false}
        disabled={!functions.length || pending}
        value={selected ? { id: selected.id, name: selected.name } : null}
        onChange={(option) => {
          if (!pending) {
            setSelectedID(option.id);
            if (attempt?.mode === 'agent')
              setAttempt({ ...attempt, functionID: option.id });
            else if (attempt) setAttempt(null);
            setError('');
          }
        }}
      >
        <Select.Button>
          {selected
            ? `${selected.appName} / ${selected.name}`
            : 'Select a function to invoke'}
        </Select.Button>
        <Select.Options className="w-full min-w-0">
          {functions.map((fn) => (
            <Select.Option
              key={fn.id}
              option={{ id: fn.id, name: fn.name, disabled: pending }}
            >
              {fn.appName} / {fn.name}
            </Select.Option>
          ))}
        </Select.Options>
      </Select>
      {!functions.length && (
        <p className="text-muted flex items-start gap-2 text-sm">
          <RiInformationLine
            aria-hidden="true"
            className="mt-0.5 h-4 w-4 shrink-0"
          />
          Your first run will appear here once the function is invoked
        </p>
      )}
      {functions.length > 0 && (
        <>
          {selected &&
            (functionError ? (
              <Alert severity="error">
                Could not load function details. Check your connection.{' '}
                <Button
                  label="Retry function details"
                  kind="secondary"
                  appearance="outlined"
                  onClick={() => refetchFunctions()}
                />
              </Alert>
            ) : !details ? (
              <p role="status" className="text-muted text-sm">
                Loading function details…
              </p>
            ) : acceptsPayload ? (
              <CodeBlock.Wrapper>
                <CodeBlock
                  copyButton={<QuietCopyButton content={payload} />}
                  tab={{
                    content: payload,
                    language: 'json',
                    readOnly: pending,
                    handleChange: setPayload,
                  }}
                  minLines={6}
                />
              </CodeBlock.Wrapper>
            ) : (
              <p className="text-muted text-sm">
                Cron functions without event triggers run without payload data.
              </p>
            ))}
          <Button
            label="Invoke function"
            aria-label="Invoke function"
            loading={invocation.isLoading || pending}
            disabled={
              (!!completion && !showSetupGuide) ||
              !!functionError ||
              !details ||
              pending ||
              invocation.isLoading
            }
            onClick={submit}
          />
        </>
      )}
      {error && <Alert severity="error">{error}</Alert>}
      {runs.isError && (
        <Alert severity="error">
          Could not load run status.{' '}
          <Button
            label="Retry status check"
            kind="secondary"
            appearance="outlined"
            onClick={() => runs.refetch()}
          />
        </Alert>
      )}
      {attempt?.functionID && candidates.length > 1 && (
        <Select
          label="Run"
          onChange={(option) => setAttempt({ ...attempt, runID: option.id })}
        >
          <Select.Button>Choose your run</Select.Button>
          <Select.Options className="w-full min-w-0">
            {candidates.map((candidate) => (
              <Select.Option
                key={candidate.id}
                option={{ id: candidate.id, name: candidate.id }}
              >
                {candidate.id} · {candidate.status}
              </Select.Option>
            ))}
          </Select.Options>
        </Select>
      )}
      {(attempt?.mode === 'agent' ||
        (attempt?.functionID && (error || runs.isError))) && (
        <div className="flex flex-wrap items-center gap-3">
          {attempt?.mode === 'agent' && (
            <p className="text-muted w-full text-sm">
              Pause your agent before switching to avoid duplicate runs.
            </p>
          )}
          <Link href="/runs?showAllRuns=true" target="_self">
            Check Runs
          </Link>
          <Button
            kind="secondary"
            appearance="outlined"
            label={
              attempt?.mode === 'agent'
                ? 'Switch to manual invocation'
                : 'I checked Runs — allow another invocation'
            }
            disabled={invocation.isLoading}
            onClick={() => {
              setAttempt(null);
              setError('');
            }}
          />
        </div>
      )}
    </section>
  );
}
