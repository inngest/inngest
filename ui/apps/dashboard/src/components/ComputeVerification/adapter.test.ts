import { describe, expect, it } from 'vitest';
import {
  ComputeVerificationStatus as Status,
  ComputeVerificationCheckKind as Kind,
  ComputeVerificationError as Failure,
} from '@/gql/graphql';
import { adaptVerification, allowsSandboxList, type Summary } from './adapter';
import { getVerificationScreen } from './state';
const account = {
  accountID: 'target',
  companyName: 'Team',
  profileID: 'user',
  displayName: 'User',
  maskedEmail: '',
};
const summary = (patch: Partial<Summary> = {}): Summary => ({
  status: Status.Required,
  canVerify: true,
  checks: [Kind.Email, Kind.Phone, Kind.Card].map((kind) => ({
    kind,
    passed: false,
    display: null,
  })),
  ...patch,
});
const adapt = (data: Summary | null, transient = {}) =>
  adaptVerification(data, account, 'target', 'api', transient);

describe('backend verification adapter', () => {
  it.each([Status.Verified, Status.Exempt])(
    'uses %s as completion without requiring checks or admin permission',
    (status) => {
      expect(
        getVerificationScreen(
          adapt(summary({ status, checks: [], canVerify: false })),
        ),
      ).toBe('complete');
    },
  );
  it('does not infer completion from three passed checks', () => {
    const allPassed = summary({
      checks: [Kind.Email, Kind.Phone, Kind.Card].map((kind) => ({
        kind,
        passed: true,
        display: 'saved',
      })),
    });
    expect(getVerificationScreen(adapt(allPassed))).toBe('finalizing');
    expect(
      getVerificationScreen(adapt(allPassed, { failure: Failure.RateLimited })),
    ).toBe('rate-limited');
  });
  it.each([
    [Status.Suspended, 'suspended'],
    [Status.NotEligible, 'early-access-unavailable'],
  ] as const)('preserves the %s account decision', (status, screen) => {
    expect(
      getVerificationScreen(
        adapt(summary({ status }), { failure: Failure.RateLimited }),
      ),
    ).toBe(screen);
  });
  it('keeps account mismatch ahead of completion', () => {
    const state = adaptVerification(
      summary({ status: Status.Verified }),
      account,
      'another-account',
      'api',
    );
    expect(getVerificationScreen(state)).toBe('mismatch');
  });
  it('does not fabricate progress after a failed read', () => {
    expect(getVerificationScreen(adapt(null))).toBe('unavailable');
  });
  it('shows admin required even with saved progress', () => {
    expect(getVerificationScreen(adapt(summary({ canVerify: false })))).toBe(
      'admin-required',
    );
  });
  it('keeps the SMS challenge local and maps incorrect and expired codes', () => {
    const data = summary({
      checks: [
        { kind: Kind.Email, passed: true, display: 'verified@example.com' },
      ],
    });
    expect(getVerificationScreen(adapt(data))).toBe('phone');
    const state = adapt(data, {
      phoneNumber: '+14155550100',
      failure: Failure.CodeIncorrect,
    });
    expect(getVerificationScreen(state)).toBe('code');
    expect(state.codeError).toBe('incorrect');
    expect(
      adapt(data, { phoneNumber: '+14155550100', failure: Failure.CodeExpired })
        .codeError,
    ).toBe('expired');
  });
  it('maps only a known missing card name to a field-specific error', () => {
    expect(
      adapt(summary(), { failure: Failure.CardNameRequired }).cardError,
    ).toBe('name-required');
    expect(adapt(summary(), { failure: Failure.CardDeclined }).cardError).toBe(
      'bank',
    );
  });
  it.each([Failure.EmailUnverified, Failure.EmailDisposable])(
    'recovers email for %s',
    (failure) => {
      expect(
        getVerificationScreen(adapt(summary(), { failure, action: 'email' })),
      ).toBe('email');
    },
  );
  it('waits for the automatic email check before showing the email screen', () => {
    expect(getVerificationScreen(adapt(summary()))).toBe('loading');
    expect(getVerificationScreen(adapt(summary(), { action: 'email' }))).toBe(
      'email',
    );
  });
});

describe('shared sandbox list/detail UI policy', () => {
  it.each([
    [Status.Required, true, false],
    [Status.Suspended, true, false],
    [Status.Verified, false, true],
    [Status.Exempt, false, true],
    [Status.NotEligible, true, true],
    [Status.NotEligible, false, false],
    [undefined, true, true],
    [undefined, false, false],
  ] as const)('handles %s with entitlement %s', (status, enabled, allowed) => {
    expect(allowsSandboxList(status, enabled)).toBe(allowed);
  });
});
