import { useCallback, useEffect, useRef, useState } from 'react';
import { useClient } from 'urql';
import type {
  ComputeVerificationQuery,
  ComputeVerificationError,
} from '@/gql/graphql';
import {
  AddVerificationCardDocument,
  CheckVerificationCardDocument,
  CheckVerificationCodeDocument,
  CheckVerificationEmailDocument,
  ComputeVerificationDocument,
  SendVerificationCodeDocument,
} from './operations';
import type { Summary, TransientVerification } from './adapter';

type Command =
  | { kind: 'email' }
  | { kind: 'send'; phoneNumber: string }
  | { kind: 'code'; phoneNumber: string; code: string }
  | { kind: 'card'; paymentMethodID: string }
  | { kind: 'finish-card'; setupIntentID: string };
type Payload = {
  verification: Summary | null;
  error: ComputeVerificationError | null;
  cardClientSecret: string | null;
};

export function useComputeVerification(targetAccountID: string) {
  const client = useClient();
  const [account, setAccount] = useState<ComputeVerificationQuery['account']>();
  const [loading, setLoading] = useState(true);
  const [readError, setReadError] = useState(false);
  const [pending, setPending] = useState(false);
  const [transient, setTransient] = useState<TransientVerification>({});
  const accountRef = useRef<ComputeVerificationQuery['account']>();
  const ready = useRef(false);
  const epoch = useRef(0);
  const busy = useRef(false);
  const emailAttempted = useRef(false);

  const load = useCallback(async () => {
    const version = epoch.current;
    ready.current = false;
    setLoading(true);
    try {
      const result = await client
        .query(
          ComputeVerificationDocument,
          {},
          { requestPolicy: 'network-only' },
        )
        .toPromise();
      if (version !== epoch.current) return;
      if (result.error || !result.data?.account.computeVerification)
        throw new Error('Verification unavailable');
      accountRef.current = result.data.account;
      ready.current = true;
      setAccount(result.data.account);
      setReadError(false);
      return result.data.account;
    } catch {
      if (version === epoch.current) setReadError(true);
    } finally {
      if (version === epoch.current) setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    void load();
    return () => {
      epoch.current += 1;
    };
  }, [load, targetAccountID]);

  useEffect(() => {
    if (!transient.cooldown) return;
    const timer = window.setTimeout(
      () =>
        setTransient((value) => ({
          ...value,
          cooldown: Math.max(0, (value.cooldown ?? 0) - 1),
        })),
      1000,
    );
    return () => window.clearTimeout(timer);
  }, [transient.cooldown]);

  async function run(command: Command): Promise<Payload | undefined> {
    const selected = accountRef.current;
    if (
      busy.current ||
      !ready.current ||
      selected?.id !== targetAccountID ||
      !selected.computeVerification?.canVerify ||
      selected.computeVerification.status !== 'REQUIRED'
    )
      return;
    busy.current = true;
    const version = epoch.current;
    setPending(true);
    setTransient((value) => ({ ...value, failure: undefined }));
    try {
      const response = await (async () => {
        switch (command.kind) {
          case 'email': {
            const result = await client
              .mutation(CheckVerificationEmailDocument, {})
              .toPromise();
            return {
              error: result.error,
              payload: result.data?.checkComputeVerificationEmail,
            };
          }
          case 'send': {
            const result = await client
              .mutation(SendVerificationCodeDocument, {
                phoneNumber: command.phoneNumber,
              })
              .toPromise();
            return {
              error: result.error,
              payload: result.data?.sendComputeVerificationCode,
            };
          }
          case 'code': {
            const result = await client
              .mutation(CheckVerificationCodeDocument, {
                phoneNumber: command.phoneNumber,
                code: command.code,
              })
              .toPromise();
            return {
              error: result.error,
              payload: result.data?.checkComputeVerificationCode,
            };
          }
          case 'card': {
            const result = await client
              .mutation(AddVerificationCardDocument, {
                paymentMethodID: command.paymentMethodID,
              })
              .toPromise();
            return {
              error: result.error,
              payload: result.data?.addComputeVerificationCard,
            };
          }
          case 'finish-card': {
            const result = await client
              .mutation(CheckVerificationCardDocument, {
                setupIntentID: command.setupIntentID,
              })
              .toPromise();
            return {
              error: result.error,
              payload: result.data?.checkComputeVerificationCard,
            };
          }
        }
      })();
      if (version !== epoch.current) return;
      if (response.error || !response.payload)
        throw new Error('Verification request failed');
      const payload = response.payload;
      if (payload.verification) {
        accountRef.current = {
          ...selected,
          computeVerification: payload.verification,
        };
        ready.current = true;
        setReadError(false);
        setAccount(accountRef.current);
      }
      setTransient((value) => ({
        ...value,
        failure: payload.error ?? undefined,
        action: command.kind === 'finish-card' ? 'card' : command.kind,
        ...(command.kind === 'send' && !payload.error
          ? { phoneNumber: command.phoneNumber, cooldown: 30 }
          : {}),
      }));
      if (!payload.verification) await load();
      return payload;
    } catch {
      if (version === epoch.current) {
        // A write may have succeeded. Read its result before offering another write.
        await load();
        if (version === epoch.current)
          setTransient((value) => ({ ...value, failure: 'NETWORK' }));
      }
    } finally {
      busy.current = false;
      if (version === epoch.current) setPending(false);
    }
  }

  useEffect(() => {
    const summary = account?.computeVerification;
    if (
      !loading &&
      !readError &&
      account?.id === targetAccountID &&
      summary?.status === 'REQUIRED' &&
      summary.canVerify &&
      !summary.checks.some((check) => check.kind === 'EMAIL' && check.passed) &&
      !emailAttempted.current
    ) {
      emailAttempted.current = true;
      void run({ kind: 'email' });
    }
  });

  async function retry() {
    const version = epoch.current;
    const latest = await load();
    if (version !== epoch.current) return;
    setTransient((value) => ({ ...value, failure: undefined }));
    // Email rechecks are safe: they finish a rate-limited finalization, and
    // they redo an automatic email check that failed.
    const checks = latest?.computeVerification?.checks ?? [];
    if (
      latest?.computeVerification?.status === 'REQUIRED' &&
      (checks.every((check) => check.passed) ||
        !checks.some((check) => check.kind === 'EMAIL' && check.passed))
    )
      await run({ kind: 'email' });
  }

  return {
    account,
    loading,
    readError,
    pending,
    transient,
    run,
    retry,
    changeNumber: () => setTransient({}),
  };
}
