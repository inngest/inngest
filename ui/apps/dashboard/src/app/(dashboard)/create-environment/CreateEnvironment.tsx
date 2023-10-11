'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import Button from '@/components/Button';
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
      <p className="my-4 max-w-[400px] text-sm font-medium text-slate-600">
        You can create a new fully isolated environment like <em>Staging</em> or <em>QA</em> for
        your developer workflow needs
      </p>
      <form onSubmit={handleSubmit} className="flex flex-col items-start gap-4">
        <Input
          className="min-w-[300px]"
          type="text"
          name="name"
          placeholder="e.g. Staging, QA"
          required
          error={error}
        />
        <Button type="submit" disabled={isDisabled}>
          Create environment
        </Button>
      </form>
    </>
  );
}
