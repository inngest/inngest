import { useEffect, useRef, useState } from 'react';
import { TooltipProvider } from '@inngest/components/Tooltip';
import { ComputeVerification } from '../ComputeVerification';
import { formatPhone } from '../state';
import type { VerificationActions, VerificationState } from '../types';
import {
  DemoNavigation,
  type demoCompanies,
  type demoProfiles,
} from './DemoNavigation';
import { MockCardFields } from './MockCardFields';
import {
  initialState,
  scenarios,
  scenarioState,
  type ScenarioID,
} from './scenarios';

function readScenario(): ScenarioID {
  const id = new URLSearchParams(window.location.search).get('scenario');
  return id && Object.hasOwn(scenarios, id) ? (id as ScenarioID) : 'happy';
}

export function VerificationDemo() {
  const [scenario, setScenario] = useState<ScenarioID>(readScenario);
  const [reset, setReset] = useState(0);
  return (
    <ScenarioDemo
      key={`${scenario}:${reset}`}
      scenario={scenario}
      onScenario={(id) => {
        const url = new URL(window.location.href);
        url.searchParams.set('scenario', id);
        window.history.replaceState(null, '', url);
        setScenario(id);
      }}
      onReset={() => setReset((value) => value + 1)}
    />
  );
}

function ScenarioDemo({
  scenario,
  onScenario,
  onReset,
}: {
  scenario: ScenarioID;
  onScenario: (id: ScenarioID) => void;
  onReset: () => void;
}) {
  const [state, setState] = useState<VerificationState>(() =>
    scenarioState(scenario),
  );
  const [populated, setPopulated] = useState(
    scenario === 'happy' || 'populatedCard' in scenarios[scenario],
  );
  const [emailManaged, setEmailManaged] = useState(false);
  const [message, setMessage] = useState('');
  const generation = useRef(0);
  useEffect(
    () => () => {
      generation.current += 1;
    },
    [],
  );
  useEffect(() => {
    const timer = window.setInterval(
      () =>
        setState((current) =>
          current.resendCooldownSeconds
            ? {
                ...current,
                resendCooldownSeconds: Math.max(
                  0,
                  current.resendCooldownSeconds - 1,
                ),
              }
            : current,
        ),
      1000,
    );
    return () => window.clearInterval(timer);
  }, []);

  async function simulate(update: Partial<VerificationState>) {
    const currentGeneration = generation.current;
    setState((current) => ({ ...current, pending: true }));
    await new Promise((resolve) => window.setTimeout(resolve, 650));
    if (generation.current !== currentGeneration) return;
    setState((current) => ({ ...current, ...update, pending: false }));
  }

  function selectContext(selected: VerificationState['selectedAccount']) {
    if (
      selected.accountID === state.selectedAccount.accountID &&
      selected.profileID === state.selectedAccount.profileID
    )
      return;
    generation.current += 1;
    setState({
      ...initialState,
      originatingAccount: state.originatingAccount,
      entrySource: state.entrySource,
      selectedAccount: selected,
    });
    setPopulated(false);
    setEmailManaged(false);
    setMessage(
      'Account context changed. Local inputs, errors, and mocked progress cleared.',
    );
  }

  const actions: VerificationActions = {
    onSendCode: (phone) =>
      simulate({
        phone: 'code-sent',
        destinationPhone: `+1 ${formatPhone(phone)}`,
        phoneError: undefined,
        codeError: undefined,
        resendCooldownSeconds: 30,
      }),
    onVerifyCode: (code) =>
      simulate(
        code === '123456'
          ? {
              phone: 'complete',
              codeError: undefined,
              verifiedPhone: '+1 ••• ••• 0132',
            }
          : { codeError: 'incorrect' },
      ),
    onResendCode: () =>
      simulate({ codeError: undefined, resendCooldownSeconds: 30 }),
    onChangeNumber: () => {
      setState((current) => ({
        ...current,
        phone: 'incomplete',
        codeError: undefined,
        resendCooldownSeconds: 0,
      }));
    },
    onSubmitCard: () =>
      simulate({
        card: 'complete',
        cardError: undefined,
        verifiedCard: 'Visa •••• 4242',
      }),
    onManageEmail: () => {
      setEmailManaged(true);
      setMessage(
        'Mock profile opened and primary email marked verified. Continue verification to proceed.',
      );
    },
    onContinueEmail: () =>
      simulate(
        emailManaged
          ? { email: 'complete', emailError: undefined }
          : {
              emailError:
                'Your primary email still needs attention. Verify or replace it in your profile.',
            },
      ),
    onSwitchCompany: () =>
      setMessage('Use the company menu to choose an organization.'),
    onSwitchProfile: () =>
      setMessage('Use the profile menu to choose an account.'),
    onRetry: () => simulate({ status: 'verification' }),
    onReturn: () => {
      setMessage(
        `Return callback called for ${state.entrySource}. No external navigation performed.`,
      );
    },
    onContactSupport: () => {
      setMessage('Contact-support callback called. No message sent.');
    },
  };

  return (
    <TooltipProvider>
      <DemoNavigation
        state={state}
        actions={actions}
        onSelectCompany={(company: (typeof demoCompanies)[number]) =>
          selectContext({ ...state.selectedAccount, ...company })
        }
        onSelectProfile={(profile: (typeof demoProfiles)[number]) =>
          selectContext({ ...state.selectedAccount, ...profile })
        }
      >
        <ComputeVerification
          state={state}
          actions={actions}
          cardReady={populated}
          renderCardFields={(props) => (
            <MockCardFields {...props} populated={populated} />
          )}
        />
      </DemoNavigation>
      <details
        data-demo-controls
        className="bg-canvasSubtle border-subtle text-basis fixed bottom-10 right-4 z-50 max-h-[80vh] w-[min(320px,calc(100vw-32px))] overflow-auto rounded-lg border p-3 text-sm shadow-lg"
      >
        <summary className="cursor-pointer font-medium">
          Development demo · scenarios
        </summary>
        <div className="mt-3 flex flex-col gap-3">
          <p className="text-muted text-xs">
            Mock only. SMS code: 123456. Card values are fixed, read-only dummy
            fixtures.
          </p>
          <label className="flex flex-col gap-1">
            Scenario
            <select
              aria-label="Scenario"
              className="border-muted bg-canvasBase rounded border p-2"
              value={scenario}
              onChange={(event) => onScenario(event.target.value as ScenarioID)}
            >
              {Object.entries(scenarios).map(([id, item]) => (
                <option key={id} value={id}>
                  {item.label}
                </option>
              ))}
            </select>
          </label>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={populated}
              onChange={(event) => setPopulated(event.target.checked)}
            />
            Populate dummy card
          </label>
          <button className="border-muted rounded border p-2" onClick={onReset}>
            Reset scenario
          </button>
          <p role="status" className="text-muted text-xs">
            {message}
          </p>
        </div>
      </details>
    </TooltipProvider>
  );
}
