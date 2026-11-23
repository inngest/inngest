// @vitest-environment jsdom
import { StrictMode, type ReactNode } from 'react';
import { act, cleanup, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  ComputeVerificationCheckKind as Kind,
  ComputeVerificationStatus as Status,
  ComputeVerificationError as Failure,
  type ComputeVerificationQuery,
} from '@/gql/graphql';
import {
  CheckVerificationEmailDocument,
  SendVerificationCodeDocument,
  CheckVerificationCodeDocument,
  CheckVerificationCardDocument,
} from './operations';
import { useComputeVerification } from './useComputeVerification';
const mocks = vi.hoisted(() => ({ query: vi.fn(), mutation: vi.fn() }));
vi.mock('urql', () => ({ useClient: () => mocks }));
const account = (): ComputeVerificationQuery['account'] => ({
  id: 'target',
  name: 'Team',
  ents: { sandboxes: { enabled: false } },
  computeVerification: {
    status: Status.Required,
    canVerify: true,
    checks: [Kind.Email, Kind.Phone, Kind.Card].map((kind) => ({
      kind,
      passed: kind === Kind.Email,
      display: null,
    })),
  },
});
const result = (value: unknown) => ({
  toPromise: () => Promise.resolve(value),
});
beforeEach(() => {
  mocks.query.mockImplementation(() =>
    result({ data: { account: account() } }),
  );
});
afterEach(() => {
  cleanup();
  vi.resetAllMocks();
});

describe('live verification controller', () => {
  it('fetches with network-only and refuses mutations for a mismatched account', async () => {
    const { result: hook } = renderHook(() => useComputeVerification('other'));
    await waitFor(() => expect(hook.current.loading).toBe(false));
    await act(() =>
      hook.current.run({ kind: 'send', phoneNumber: '+14155550100' }),
    );
    expect(mocks.query.mock.calls[0][2]).toEqual({
      requestPolicy: 'network-only',
    });
    expect(mocks.mutation).not.toHaveBeenCalled();
  });
  it('does not bootstrap email or mutate for members', async () => {
    const data = account();
    data.computeVerification!.canVerify = false;
    data.computeVerification!.checks = [];
    mocks.query.mockReturnValue(result({ data: { account: data } }));
    const { result: hook } = renderHook(() => useComputeVerification('target'));
    await waitFor(() => expect(hook.current.loading).toBe(false));
    await act(() => hook.current.run({ kind: 'email' }));
    expect(mocks.mutation).not.toHaveBeenCalled();
  });
  it('bootstraps missing email once under StrictMode and does not loop on rejection', async () => {
    const data = account();
    data.computeVerification!.checks = [];
    mocks.query.mockReturnValue(result({ data: { account: data } }));
    mocks.mutation.mockReturnValue(
      result({
        data: {
          checkComputeVerificationEmail: {
            verification: data.computeVerification,
            error: Failure.EmailUnverified,
            cardClientSecret: null,
          },
        },
      }),
    );
    const { result: hook, rerender } = renderHook(
      () => useComputeVerification('target'),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <StrictMode>{children}</StrictMode>
        ),
      },
    );
    await waitFor(() =>
      expect(hook.current.transient.failure).toBe(Failure.EmailUnverified),
    );
    rerender();
    expect(mocks.mutation).toHaveBeenCalledTimes(1);
    expect(mocks.mutation.mock.calls[0][0]).toBe(
      CheckVerificationEmailDocument,
    );
  });
  it('sends and checks the same E164 number with the exact typed operations', async () => {
    const data = account();
    mocks.mutation.mockImplementation((document) =>
      result({
        data:
          document === SendVerificationCodeDocument
            ? {
                sendComputeVerificationCode: {
                  verification: data.computeVerification,
                  error: null,
                  cardClientSecret: null,
                },
              }
            : {
                checkComputeVerificationCode: {
                  verification: data.computeVerification,
                  error: Failure.CodeIncorrect,
                  cardClientSecret: null,
                },
              },
      }),
    );
    const { result: hook } = renderHook(() => useComputeVerification('target'));
    await waitFor(() => expect(hook.current.loading).toBe(false));
    await act(() =>
      hook.current.run({ kind: 'send', phoneNumber: '+14155550100' }),
    );
    expect(hook.current.transient.phoneNumber).toBe('+14155550100');
    expect(hook.current.transient.cooldown).toBe(30);
    await act(() =>
      hook.current.run({
        kind: 'code',
        phoneNumber: hook.current.transient.phoneNumber!,
        code: '000000',
      }),
    );
    expect(mocks.mutation.mock.calls[1]).toEqual([
      CheckVerificationCodeDocument,
      { phoneNumber: '+14155550100', code: '000000' },
    ]);
    expect(hook.current.transient.failure).toBe(Failure.CodeIncorrect);
  });
  it('refreshes after a nullable mutation summary without repeating the write', async () => {
    mocks.mutation.mockReturnValue(
      result({
        data: {
          sendComputeVerificationCode: {
            verification: null,
            error: null,
            cardClientSecret: null,
          },
        },
      }),
    );
    const { result: hook } = renderHook(() => useComputeVerification('target'));
    await waitFor(() => expect(hook.current.loading).toBe(false));
    await act(() =>
      hook.current.run({ kind: 'send', phoneNumber: '+14155550100' }),
    );
    expect(mocks.query).toHaveBeenCalledTimes(2);
    expect(mocks.mutation).toHaveBeenCalledTimes(1);
  });
  it('finishes a supplied SetupIntent through the check mutation', async () => {
    const data = account();
    mocks.mutation.mockReturnValue(
      result({
        data: {
          checkComputeVerificationCard: {
            verification: {
              ...data.computeVerification,
              status: Status.Verified,
            },
            error: null,
            cardClientSecret: null,
          },
        },
      }),
    );
    const { result: hook } = renderHook(() => useComputeVerification('target'));
    await waitFor(() => expect(hook.current.loading).toBe(false));
    await act(() =>
      hook.current.run({ kind: 'finish-card', setupIntentID: 'seti_test' }),
    );
    expect(mocks.mutation.mock.calls[0]).toEqual([
      CheckVerificationCardDocument,
      { setupIntentID: 'seti_test' },
    ]);
    expect(hook.current.account?.computeVerification?.status).toBe(
      Status.Verified,
    );
  });
  it('can finalize on the first retry after a failed read', async () => {
    const data = account();
    data.computeVerification!.checks = data.computeVerification!.checks.map(
      (check) => ({ ...check, passed: true }),
    );
    mocks.query
      .mockReturnValueOnce(result({ error: new Error('offline') }))
      .mockReturnValue(result({ data: { account: data } }));
    mocks.mutation.mockReturnValue(
      result({
        data: {
          checkComputeVerificationEmail: {
            verification: {
              ...data.computeVerification,
              status: Status.Verified,
            },
            error: null,
            cardClientSecret: null,
          },
        },
      }),
    );
    const { result: hook } = renderHook(() => useComputeVerification('target'));
    await waitFor(() => expect(hook.current.readError).toBe(true));
    await act(() => hook.current.retry());
    expect(mocks.mutation).toHaveBeenCalledTimes(1);
    expect(hook.current.account?.computeVerification?.status).toBe(
      Status.Verified,
    );
    expect(hook.current.readError).toBe(false);
  });

  it('ignores a read that resolves after unmount', async () => {
    let resolve!: (value: unknown) => void;
    mocks.query.mockReturnValue({
      toPromise: () =>
        new Promise((done) => {
          resolve = done;
        }),
    });
    const { unmount } = renderHook(() => useComputeVerification('target'));
    unmount();
    await act(async () => resolve({ data: { account: account() } }));
    expect(mocks.mutation).not.toHaveBeenCalled();
  });
});
