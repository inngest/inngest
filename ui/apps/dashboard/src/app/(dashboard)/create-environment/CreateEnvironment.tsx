'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import AppLink from '@/components/AppLink';
import Input from '@/components/Forms/Input';
import { graphql } from '@/gql';

const CreateEnvironmentDocument = graphql(`
  mutation CreateEnvironment($name: String!) {
    createWorkspace(input: { name: $name }) {
      id
    }
  }
`);

export default function CreateEnvironment({}) {
  const router = useRouter();
  const [, createEnvironment] = useMutation(CreateEnvironmentDocument);
  const [isDisabled, setDisabled] = useState(false);
  const [error, setError] = useState('');

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setDisabled(true);

    const form = new FormData(event.currentTarget);
    const name = form.get('name') as string | null;
    if (!name) {
      return;
    }

    const result = await createEnvironment({ name });
    setDisabled(false);
    if (result.error) {
      toast.error(`Failed to create new environment`);
      setError('Failed to create a new environment');
      return;
    }
    toast.success(`Created new environment`);
    router.push('/env');
  }

  return (
    <>
      <h2 className="text-lg font-medium text-slate-800">Create an Environment</h2>
      <p className="my-4 max-w-[600px] text-sm font-medium text-slate-600">
        Create a shared, non-production environment like staging, QA, or canary.
      </p>
      <div className="my-4 rounded-lg bg-slate-50 px-6 py-4">
        <p className="mb-4 max-w-[600px] text-sm font-medium text-slate-600">
          Some key things to know about custom environments:
        </p>
        <ul className="ml-8 flex list-disc flex-col gap-2 text-sm font-medium text-slate-600">
          <li>
            Each environment has its own keys, event history, and functions. All data is isolated
            within each environment.
          </li>
          <li>
            You can deploy multiple apps to the same environment to fully simulate your production
            environment.
          </li>
          <li>You can create as many environments as you need.</li>
        </ul>
        <p className="mt-4 max-w-[600px] text-sm font-medium text-slate-600">
          <AppLink
            href="https://www.inngest.com/docs/platform/environments#custom-environments"
            target="_blank"
          >
            Read the docs to learn more
          </AppLink>
        </p>
      </div>
      <form onSubmit={handleSubmit} className="my-8 flex flex-row items-start gap-4">
        <Input
          className="min-w-[300px]"
          type="text"
          name="name"
          placeholder="e.g. Staging, QA"
          required
          error={error}
        />
        <Button kind="primary" type="submit" disabled={isDisabled} label="Create Environment" />
      </form>
      <div className="mt-16 flex">
        <Button
          href="/env"
          kind="default"
          appearance="outlined"
          label="← Back to all environments"
        />
      </div>
    </>
  );
}
