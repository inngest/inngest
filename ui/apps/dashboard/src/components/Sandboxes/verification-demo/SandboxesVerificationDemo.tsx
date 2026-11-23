import { useState } from 'react';
import { Button } from '@inngest/components/Button';

import SandboxesEmptyState from '../SandboxesEmptyState';
import type { SandboxesVerificationState } from '../SandboxesVerificationCallout';

const scenarios = {
  free: {
    label: 'Not eligible · upgrade only',
    state: {
      status: 'required',
      canVerify: true,
      started: false,
      progress: { completed: 0, total: 2 },
    },
  },
  start: {
    label: 'Eligible · not started',
    state: {
      status: 'required',
      canVerify: true,
      started: false,
      progress: { completed: 0, total: 2 },
    },
  },
  resume: {
    label: 'Eligible · phone verified · resume card',
    state: {
      status: 'required',
      canVerify: true,
      started: true,
      progress: { completed: 1, total: 2 },
    },
  },
  started: {
    label: 'Eligible · started · no checks complete',
    state: {
      status: 'required',
      canVerify: true,
      started: true,
      progress: { completed: 0, total: 2 },
    },
  },
  member: {
    label: 'Eligible · admin required',
    state: { status: 'required', canVerify: false },
  },
  loading: { label: 'Eligible · loading', state: { status: 'loading' } },
  error: {
    label: 'Eligible · status error · retry',
    state: { status: 'error' },
  },
  verified: {
    label: 'Verified · no gate',
    state: { status: 'verified' },
  },
  exempt: {
    label: 'Exempt · no gate',
    state: { status: 'not-required' },
  },
} satisfies Record<
  string,
  { label: string; state: SandboxesVerificationState }
>;

type Scenario = keyof typeof scenarios;

function readScenario(): Scenario {
  const value = new URLSearchParams(window.location.search).get('scenario');
  return value && Object.hasOwn(scenarios, value)
    ? (value as Scenario)
    : 'start';
}

export function SandboxesVerificationDemo() {
  const [scenario, setScenario] = useState<Scenario>(readScenario);
  const [handoff, setHandoff] = useState(false);

  function selectScenario(next: Scenario) {
    const url = new URL(window.location.href);
    url.searchParams.set('scenario', next);
    window.history.replaceState(null, '', url);
    setScenario(next);
    setHandoff(false);
  }

  return (
    <div className="bg-canvasBase text-basis min-h-screen">
      <header className="border-subtle bg-canvasSubtle flex flex-wrap items-center justify-between gap-3 border-b px-6 py-3 text-sm">
        <p>Development preview · verification entry only</p>
        <label className="flex min-w-0 flex-wrap items-center gap-2">
          Scenario
          <select
            className="border-muted bg-canvasBase max-w-full rounded border px-2 py-1"
            value={scenario}
            onChange={(event) => selectScenario(event.target.value as Scenario)}
          >
            {Object.entries(scenarios).map(([id, item]) => (
              <option key={id} value={id}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      </header>
      <main>
        {handoff ? (
          <section
            aria-labelledby="handoff-title"
            className="border-subtle mx-auto my-16 flex w-[calc(100%-48px)] max-w-lg flex-col gap-4 rounded-lg border p-6"
          >
            <h1 id="handoff-title" className="text-xl font-medium">
              Verification flow handoff
            </h1>
            <p className="text-muted text-sm">
              The entry callback has fired. The production host will open the
              existing verification flow with this context.
            </p>
            <dl className="text-sm">
              <dt className="text-muted">Account</dt>
              <dd>preview-account</dd>
              <dt className="text-muted mt-2">Return path</dt>
              <dd>/env/production/sandboxes</dd>
              <dt className="text-muted mt-2">Action</dt>
              <dd>
                {scenario === 'start'
                  ? 'Start verification'
                  : 'Resume verification'}
              </dd>
            </dl>
            <Button
              label="Simulate completion and return"
              onClick={() => selectScenario('verified')}
            />
            <Button
              kind="secondary"
              appearance="outlined"
              label="Return without completing"
              onClick={() => setHandoff(false)}
            />
          </section>
        ) : (
          <SandboxesEmptyState
            requiresUpgrade={scenario === 'free'}
            verification={{
              state: scenarios[scenario].state,
              onVerify: () => setHandoff(true),
              onRetry: () => selectScenario('start'),
            }}
          />
        )}
      </main>
    </div>
  );
}
