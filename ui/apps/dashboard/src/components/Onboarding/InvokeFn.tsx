import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { NewButton } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock/CodeBlock';
import { NewLink } from '@inngest/components/Link';
import { Select } from '@inngest/components/Select/Select';

import { pathCreator } from '@/utils/urls';
import { OnboardingSteps } from '../Onboarding/types';
import useOnboardingStep from './useOnboardingStep';

/* TODO: fetch list of functions action */
const functions = [
  {
    id: '1',
    name: 'Function 1',
  },
  {
    id: '2',
    name: 'Function 2',
  },
];

export default function InvokeFn() {
  const { updateLastCompletedStep } = useOnboardingStep();
  const [selectedFunction, setSelectedFunction] = useState(functions[0]);
  const router = useRouter();

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        You can send a test event and see your function in action. You will be able to access all
        our monitoring and debugging features.{' '}
        <NewLink
          className="inline-block"
          size="small"
          href="https://www.inngest.com/docs/features/events-triggers?ref=app-onboarding-invoke-fn"
        >
          Read more
        </NewLink>
      </p>
      <div className="border-subtle my-6 rounded-sm border px-6 py-4">
        <p className="text-muted mb-2 text-sm font-medium">Select function to test:</p>
        <Select
          onChange={setSelectedFunction}
          isLabelVisible={false}
          label="Select function"
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
              content: JSON.stringify(
                {
                  data: {
                    example: 'type a JSON payload here to test your function',
                  },
                },
                null,
                2
              ),
              readOnly: false,
              language: 'json',
            }}
          />
        </CodeBlock.Wrapper>
        <div className="mt-6 flex items-center gap-2">
          <NewButton
            label="Invoke test function"
            onClick={() => {
              updateLastCompletedStep(OnboardingSteps.InvokeFn);
              {
                /* TODO: add invoke action */
              }
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
