'use client';

import * as process from 'process';
import { useState } from 'react';
import { type Route } from 'next';
import { useRouter, useSearchParams } from 'next/navigation';
import { CheckCircleIcon, ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';

import AppLink from '@/components/AppLink';
import Input from '@/components/Forms/Input';
import InngestLogo from '@/icons/InngestLogo';
import SplitView from '../../SplitView';

const minPasswordLength = 8;

export default function PasswordResetComplete() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [isLoading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState<boolean>(false);

  async function handleSubmit(event: React.SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError('');

    const token = searchParams.get('token');
    const password = event.currentTarget.password.value;
    const confirmPassword = event.currentTarget.confirmPassword.value;

    if (!token) {
      setLoading(false);
      setError('Password reset token missing from URL, please check the link in your email');
      return;
    }

    if (password.length < minPasswordLength) {
      setLoading(false);
      setError(`Password must be at least ${minPasswordLength} characters`);
      return;
    }

    if (password !== confirmPassword) {
      setLoading(false);
      setError('Passwords do not match, please try again');
      return;
    }

    const result = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/reset-password/reset`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'content-type': 'application/json',
      },
      body: JSON.stringify({
        password,
        token,
      }),
    });

    let error;
    try {
      const json = await result?.json();
      error = json?.error;
    } catch (e) {}

    setLoading(false);
    if (result.status > 299) {
      setError(error || 'There was an error attempting to reset your password');
    } else {
      // Show the success notification to tell the user to sign in now
      setSuccess(true);
    }
  }

  return (
    <SplitView>
      <div className="mx-auto my-8 w-80 text-center">
        <InngestLogo className="mb-8 flex justify-center" />
        <h1 className="text-lg font-semibold text-slate-800">Complete Your Password Reset</h1>
        <p className="mt-4 text-center text-sm font-medium text-slate-700">
          Set your new password:
        </p>
        <form className="my-5 flex flex-col gap-4 text-center" onSubmit={handleSubmit}>
          <Input
            size="lg"
            type="password"
            name="password"
            placeholder="Your new password"
            required
          />
          <Input
            size="lg"
            type="password"
            name="confirmPassword"
            placeholder="Confirm new password"
            required
          />
          <Button
            kind="primary"
            type="submit"
            disabled={isLoading}
            size="large"
            label={isLoading ? 'Completing Password Reset...' : 'Complete Password Reset'}
          />
          {error && (
            <div className="my-4 flex rounded-md border border-red-600 bg-red-100 p-4 text-left text-sm text-red-600">
              <ExclamationCircleIcon className="mr-2 max-h-5 text-red-600" />
              {error}
            </div>
          )}
          {success && (
            <a
              href={process.env.NEXT_PUBLIC_SIGN_IN_PATH as Route}
              className="my-4 flex rounded-md border border-green-600 bg-green-100 p-4 text-center text-sm text-green-600"
            >
              <div className="flex justify-center pb-4">
                <CheckCircleIcon className="mr-2 max-h-5 text-green-600" />
                Your password has been updated.{' '}
              </div>
              <AppLink
                href={process.env.NEXT_PUBLIC_SIGN_IN_PATH as Route}
                label="Please click here to sign in with your new password"
              />
              .
            </a>
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
