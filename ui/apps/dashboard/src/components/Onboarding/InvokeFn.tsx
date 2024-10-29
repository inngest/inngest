import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert } from '@inngest/components/Alert/Alert';
import { NewButton } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock/CodeBlock';
import { parseCode } from '@inngest/components/InvokeButton/utils';
import { NewLink } from '@inngest/components/Link';
import { Select } from '@inngest/components/Select/Select';

import { pathCreator } from '@/utils/urls';
import { type EntityType } from '../Metrics/Dashboard';
import { OnboardingSteps } from '../Onboarding/types';
import { invokeFunction, prefetchFunctions } from './actions';
import useOnboardingStep from './useOnboardingStep';

const initialCode = JSON.stringify(
  {
    data: {
      example: 'type a JSON payload here to test your function',
    },
  },
  null,
  2
);

export default function InvokeFn() {
  const { updateLastCompletedStep } = useOnboardingStep();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [functions, setFunctions] = useState<EntityType[]>([]);
  const [selectedFunction, setSelectedFunction] = useState<EntityType | undefined>();
  const [rawPayload, setRawPayload] = useState(initialCode);
  const router = useRouter();

  useEffect(() => {
    const loadFunctions = async () => {
      try {
        setLoading(true);
        const fetchedFunctions = await prefetchFunctions();
        setFunctions(fetchedFunctions);

        // // Set the initial selected function
        // if (fetchedFunctions.length > 0) {
        //   setSelectedFunction(fetchedFunctions[0]);
        // }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load functions');
      } finally {
        setLoading(false);
      }
    };

    loadFunctions();
  }, []);
  console.log(functions);
  // const doesFunctionAcceptPayload =
  //   fn?.current?.triggers.some((trigger) => {
  //     return trigger.eventName;
  //   }) ?? false;

  const handleInvokeFn = async () => {
    if (!selectedFunction || !selectedFunction.slug) return;
    try {
      const { success } = await invokeFunction({
        functionSlug: selectedFunction.slug,
        user: parseCode(rawPayload).user,
        data: parseCode(rawPayload).data,
      });
      if (success) {
        updateLastCompletedStep(OnboardingSteps.InvokeFn);
        setError(undefined);
      } else {
        // setError()
      }
    } catch (err) {
    } finally {
    }
  };

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        You can send a test event and see your function in action. You will be able to access all
        our monitoring and debugging features.{' '}
        <NewLink
          className="inline-block"
          size="small"
          href="https://www.inngest.com/docs/features/events-triggers?ref=app-onboarding-invoke-fn"
          target="_blank"
        >
          Read more
        </NewLink>
      </p>
      <div className="border-subtle my-6 rounded-sm border px-6 py-4">
        <p className="text-muted mb-2 text-sm font-medium">Select function to test:</p>
        <Select
          onChange={setSelectedFunction}
          isLabelVisible={false}
          label={loading ? 'Loading functions...' : 'Select function'}
          multiple={false}
          value={selectedFunction}
          className="mb-6"
        >
          <Select.Button>
            <div className="text-sm font-medium leading-tight">
              {selectedFunction?.name || 'Select function'}
            </div>
          </Select.Button>
          <Select.Options className="w-full">
            {functions.map((option) => {
              return (
                <Select.Option key={option.id} option={option}>
                  {option.name}
                </Select.Option>
              );
            })}
          </Select.Options>
        </Select>
        <CodeBlock.Wrapper>
          <CodeBlock
            header={{ title: 'Invoke function' }}
            tab={{
              content: rawPayload,
              readOnly: false,
              language: 'json',
              handleChange: setRawPayload,
            }}
          />
        </CodeBlock.Wrapper>
        {error && (
          <Alert className="mt-6" severity="error">
            {error}
          </Alert>
        )}
        <div className="mt-6 flex items-center gap-2">
          <NewButton
            label="Invoke test function"
            onClick={() => {
              handleInvokeFn();
            }}
          />
          {/* TODO: add tracking */}
          <NewButton
            appearance="outlined"
            label="Skip, take me to dashboard"
            onClick={() => {
              updateLastCompletedStep(OnboardingSteps.InvokeFn);
              router.push(pathCreator.apps({ envSlug: 'production' }));
            }}
          />
        </div>
      </div>
    </div>
  );
}
