'use client';

import { useState } from 'react';
import { useUser } from '@clerk/nextjs';
import { Button } from '@inngest/components/Button';
import * as Sentry from '@sentry/nextjs';

import { Alert } from '@/components/Alert';
import { SelectInput } from '@/components/Forms/SelectInput';
import { Textarea } from '@/components/Forms/Textarea';
import { type RequestBody } from '../api/support-tickets/route';
import {
  DEFAULT_BUG_SEVERITY_LEVEL,
  formOptions,
  severityOptions,
  type BugSeverity,
  type TicketType,
} from './ticketOptions';

const instructions: { [K in Exclude<TicketType, null>]: string } = {
  bug: 'Please include any relevant run IDs, function names, event IDs in your message',
  demo: 'Please include relevant info like your use cases, estimated volume or specific needs',
  billing: 'What is your issue?',
  feature: `What's your idea?`,
  security: 'Please detail your concern',
  question: 'What would you like to know?',
};

type SupportFormProps = {
  isEnterprise: boolean;
  isPaid: boolean;
};

export function SupportForm({ isEnterprise = false, isPaid = false }: SupportFormProps) {
  const [ticketType, setTicketType] = useState<TicketType>(null);
  const [body, setBody] = useState<string>('');

  const [bugSeverity, setBugSeverityLevel] = useState<BugSeverity>(DEFAULT_BUG_SEVERITY_LEVEL);
  const [isFetching, setIsFetching] = useState<boolean>(false);
  const [result, setResult] = useState<{ ok?: boolean; message?: string }>({});
  const { user, isSignedIn } = useUser();

  const availableSeverityOptions = severityOptions.map((o) => ({
    ...o,
    label: (
      <>
        {o.label}{' '}
        {o.enterpriseOnly ? (
          <Label text="Enterprise Plan" />
        ) : o.paidOnly ? (
          <Label text="All Paid Plans" />
        ) : null}
      </>
    ),
    disabled: o.enterpriseOnly ? !isEnterprise : o.paidOnly ? !isPaid : false,
  }));

  function clearForm() {
    setTicketType(null);
    setBody('');
    setBugSeverityLevel(DEFAULT_BUG_SEVERITY_LEVEL);
    setIsFetching(false);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!ticketType) {
      return;
    }
    setIsFetching(true);

    try {
      // The following exceptions aren't user errors and should never happen.
      if (!isSignedIn) throw new Error('User must be signed in to create a support ticket.');
      if (!user.primaryEmailAddress) throw new Error('User must have a primary email address.');
      if (!user.externalId) throw new Error('User must have an external ID.');
      if (!user.publicMetadata.accountID || typeof user.publicMetadata.accountID !== 'string') {
        throw new Error('User is missing an account ID.');
      }

      const reqBody: RequestBody = {
        user: {
          id: user.externalId,
          email: user.primaryEmailAddress.emailAddress,
          name: user.fullName ?? undefined,
          accountId: user.publicMetadata.accountID,
        },
        ticket: {
          type: ticketType,
          body,
          severity: bugSeverity,
        },
      };

      const result = await fetch('/api/support-tickets', {
        method: 'POST',
        credentials: 'include',
        redirect: 'error',
        body: JSON.stringify(reqBody),
      });
      if (result.ok) {
        clearForm();
        setResult({
          ok: true,
          message: 'Support ticket created!',
        });
      } else {
        setIsFetching(false);
        setResult({
          ok: false,
          message:
            'Failed to create support ticket - please email hello@inngest.com if the problem persists',
        });
      }
    } catch (error) {
      Sentry.captureException(error);
      setIsFetching(false);
      setResult({
        ok: false,
        message:
          'Failed to create support ticket - please email hello@inngest.com if the problem persists',
      });
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex w-full flex-col gap-4">
      <label className="flex w-full flex-col gap-2 font-medium">
        What do you need help with?
        <SelectInput
          value={ticketType}
          options={formOptions}
          onChange={setTicketType}
          placeholder="A bug, request a demo, etc."
          required
        />
      </label>
      <label className="flex w-full flex-col gap-2 font-medium">
        {ticketType && instructions[ticketType]}
        <Textarea
          placeholder="Describe your issue..."
          value={body}
          onChange={setBody}
          rows={5}
          required
        />
      </label>
      {ticketType === 'bug' && (
        <label className="flex w-full flex-col gap-2 font-medium">
          How severe is your issue?
          <SelectInput
            value={bugSeverity}
            options={availableSeverityOptions}
            onChange={setBugSeverityLevel}
            placeholder="How severe is your issue?"
          />
        </label>
      )}
      <Button type="submit" disabled={isFetching} label="Create Support Ticket" kind="primary" />
      {result.message && <Alert severity={result.ok ? 'info' : 'error'}>{result.message}</Alert>}
      <p className="mt-4 text-sm">
        {isPaid ? (
          <>
            Our team will respond via email as soon as possible based on the severity of your issue.
          </>
        ) : (
          <>
            Upgrade to a paid plan to specify the severity of your issue to get faster responses and
            include colleagues via email.
          </>
        )}
      </p>
    </form>
  );
}

function Label({ text }: { text: string }) {
  return (
    <span className="ml-1 inline-flex items-center rounded px-[5px] py-0.5 text-[12px] font-semibold leading-tight text-indigo-500 ring-1 ring-inset ring-indigo-300">
      {text}
    </span>
  );
}
