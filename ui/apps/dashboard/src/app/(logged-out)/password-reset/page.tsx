'use client';

import process from 'process';
import { useState } from 'react';
import { type Route } from 'next';
import { CheckCircleIcon, ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import AppLink from '@/components/AppLink';
import Input from '@/components/Forms/Input';
import InngestLogo from '@/icons/InngestLogo';
import SplitView from '../SplitView';

export default function PasswordReset() {
  const [isLoading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState<string>('');

  async function handleSubmit(event: React.SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError('');

    const email = event.currentTarget.email.value;
    const result = await fetch(
      `${process.env.NEXT_PUBLIC_API_URL}/v1/reset-password/new?email=${encodeURIComponent(email)}`,
      {
        method: 'POST',
        credentials: 'include',
        headers: {
          'content-type': 'application/json',
        },
      }
    );

    setLoading(false);
    if (result.status >= 299) {
      setError('There was an error attempting to reset your password');
    } else {
      setSuccess(`We'll send you a link to reset your password shortly`);
    }
  }

  return (
    <SplitView>
      <div className="mx-auto my-8 w-80 text-center">
        <InngestLogo className="mb-8 flex justify-center" />
        <h1 className="text-lg font-semibold text-slate-800">Request a password reset</h1>
        <form className="my-5 flex flex-col gap-4 text-center" onSubmit={handleSubmit}>
          <Input size="lg" type="text" name="email" placeholder="Email" required />
          <Button
            type="submit"
            disabled={isLoading}
            size="large"
            label={isLoading ? 'Requesting Password Reset...' : 'Request Password Reset'}
          />
          {error && (
            <div className="my-4 flex rounded-md border border-red-600 bg-red-100 p-4 text-left text-sm text-red-600">
              <ExclamationCircleIcon className="mr-2 max-h-5 text-red-600" />
              {error}
            </div>
          )}
          {success && (
            <div className="my-4 flex rounded-md border border-green-600 bg-green-100 p-4 text-left text-sm text-green-600">
              <CheckCircleIcon className="mr-2 max-h-5 text-green-600" />
              {success}
            </div>
          )}
          <p className="mt-4 text-center text-sm font-medium text-slate-700">
            <AppLink href={process.env.NEXT_PUBLIC_SIGN_IN_PATH as Route} label="Back to sign in" />
          </p>
          <p className="text-center text-sm font-medium text-slate-700">
            Don&apos;t have an account?{' '}
            <AppLink href={process.env.NEXT_PUBLIC_SIGN_UP_PATH as Route} label="Register here" />
          </p>
        </form>
      </div>
    </SplitView>
  );
}
