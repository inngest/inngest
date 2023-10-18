'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import Input from '@/components/Forms/Input';
import Modal from '@/components/Modal';
import { graphql } from '@/gql';

const mutation = graphql(`
  mutation CreateUser($input: NewUser!) {
    createUser(input: $input) {
      id
    }
  }
`);

type FormValues = {
  email: string;
  name: string;
};

type FormErrors = { [key in keyof FormValues]?: string };

export function UserCreator() {
  const [errors, setErrors] = useState<FormErrors>({});
  const [isModalOpen, setIsModalOpen] = useState(false);
  const router = useRouter();
  const [, createUser] = useMutation(mutation);

  function cancel() {
    setIsModalOpen(false);
    setErrors({});
  }

  async function handleSubmit(event: React.SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();

    const newErrors: typeof errors = {};
    const email = event.currentTarget?.email?.value;

    // @ts-expect-error "name" is a reserved property that we're overriding, so
    // TS thinks it's a string instead of an HTML element.
    const name = event.currentTarget?.name?.value;

    if (!email) {
      newErrors.email = 'Required';
    }

    if (!name) {
      newErrors.name = 'Required';
    }

    setErrors(newErrors);
    const hasErrors = Object.keys(newErrors).length > 0;
    if (hasErrors) {
      return;
    }

    const input = {
      email,
      name,
    };

    const res = await createUser({ input });
    if (res.error) {
      toast.error('Failed to create user');
      return;
    }

    router.refresh();
    toast.success('User created');
    setIsModalOpen(false);
  }

  return (
    <>
      <Button btnAction={() => setIsModalOpen(true)} kind="primary" label="Add User" />

      <Modal className="flex w-96 flex-col gap-4" isOpen={isModalOpen} onClose={cancel}>
        <h2 className="text-lg font-medium">Create a user</h2>

        <form onSubmit={handleSubmit}>
          <Input className="mb-4" error={errors.name} label="Name" name="name" required />

          <Input
            className="mb-4"
            error={errors.email}
            label="Email"
            name="email"
            required
            type="email"
          />

          <div className="flex flex-row justify-end gap-4">
            <Button btnAction={cancel} appearance="outlined" label="Cancel" />
            <Button type="submit" label="Yes" kind="primary" />
          </div>
        </form>
      </Modal>
    </>
  );
}
