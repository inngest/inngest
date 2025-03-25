'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Link } from '@inngest/components/Link';
import { toast } from 'sonner';
import { useMutation } from 'urql';

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
    <div className="my-8">
      <h2 className="mb-2 text-2xl">Create an Environment</h2>
      <p className="text-muted max-w-[600px] text-sm">
        Create a shared, non-production environment like staging, QA, or canary.
      </p>
      <div className="bg-canvasSubtle my-6 rounded-md px-6 py-4">
        <p className="mb-4 max-w-[600px] text-sm">
          Some key things to know about custom environments:
        </p>
        <ul className="ml-8 flex list-disc flex-col gap-2 text-sm">
          <li>
            Each environment has its own keys, event history, and functions. All data is isolated
            within each environment.
          </li>
          <li>
            You can sync multiple apps to the same environment to fully simulate your production
            environment.
          </li>
          <li>You can create as many environments as you need.</li>
        </ul>
        <p className="mt-4 max-w-[600px] text-sm">
          <Link
            size="small"
            href="https://www.inngest.com/docs/platform/environments#custom-environments"
          >
            Read the docs to learn more
          </Link>
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
        <Button kind="primary" type="submit" disabled={isDisabled} label="Create environment" />
      </form>
    </div>
  );
}
