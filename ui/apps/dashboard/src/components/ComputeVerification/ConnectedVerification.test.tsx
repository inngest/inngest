// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  ComputeVerificationCheckKind as Kind,
  ComputeVerificationStatus as Status,
  ComputeVerificationError as Failure,
  type ComputeVerificationQuery,
} from '@/gql/graphql';
import { ConnectedVerification } from './ConnectedVerification';
import {
  AddVerificationCardDocument,
  CheckVerificationCodeDocument,
  CheckVerificationEmailDocument,
} from './operations';
const mocks = vi.hoisted(() => ({
  query: vi.fn(),
  mutation: vi.fn(),
  rateLimited: false,
}));
vi.mock('urql', () => ({ useClient: () => mocks }));
vi.mock('@clerk/tanstack-react-start', () => ({
  useOrganizationList: () => ({ userMemberships: { data: [] } }),
}));
vi.mock('./VerificationPage', async () => {
  const { ComputeVerification } = await import('./ComputeVerification');
  return { VerificationPage: ComputeVerification };
});
vi.mock('./StripeVerification', () => ({
  StripeVerification: ({
    controller,
    children,
  }: {
    controller: { run: (command: unknown) => Promise<unknown> };
    children: (adapter: unknown) => React.ReactNode;
  }) =>
    children({
      cardReady: true,
      renderCardFields: () => <div>Hosted card fields</div>,
      onSubmitCard: async () => {
        await controller.run({ kind: 'card', paymentMethodID: 'pm_test' });
      },
    }),
}));
const profile = {
  isMarketplace: false,
  marketplace: null,
  displayName: 'User',
  orgName: 'Team',
  orgProfilePic: null,
  userProfilePic: null,
};
let account: ComputeVerificationQuery['account'];
beforeEach(() => {
  account = {
    id: 'target',
    name: 'Team',
    ents: { sandboxes: { enabled: false } },
    computeVerification: {
      status: Status.Required,
      canVerify: true,
      checks: [Kind.Email, Kind.Phone, Kind.Card].map((kind) => ({
        kind,
        passed: false,
        display: null,
      })),
    },
  };
  mocks.query.mockImplementation(() => ({
    toPromise: async () => ({ data: { account: structuredClone(account) } }),
  }));
  mocks.mutation.mockImplementation((document) => ({
    toPromise: async () => {
      const name = document.definitions.find(
        (definition: { kind: string }) =>
          definition.kind === 'OperationDefinition',
      ).selectionSet.selections[0].name.value;
      const kind =
        document === CheckVerificationEmailDocument
          ? Kind.Email
          : document === CheckVerificationCodeDocument
          ? Kind.Phone
          : document === AddVerificationCardDocument
          ? Kind.Card
          : undefined;
      if (kind)
        account.computeVerification!.checks.find(
          (check) => check.kind === kind,
        )!.passed = true;
      const complete = account.computeVerification!.checks.every(
        (check) => check.passed,
      );
      if (complete && !mocks.rateLimited)
        account.computeVerification!.status = Status.Verified;
      return {
        data: {
          [name]: {
            verification: structuredClone(account.computeVerification),
            error: complete && mocks.rateLimited ? Failure.RateLimited : null,
            cardClientSecret: null,
          },
        },
      };
    },
  }));
});
afterEach(() => {
  cleanup();
  vi.resetAllMocks();
  mocks.rateLimited = false;
});
function setup(target = 'target') {
  render(
    <ConnectedVerification
      account_id={target}
      source="api"
      profile={profile}
      profileID="user"
      onManageEmail={vi.fn()}
    />,
  );
}
async function verifyPhone() {
  await screen.findByRole('heading', { name: 'Verify your phone' });
  fireEvent.change(screen.getByLabelText('Mobile number'), {
    target: { value: '4155550100' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Send code' }));
  await screen.findByRole('heading', { name: 'Enter your verification code' });
  fireEvent.paste(screen.getByLabelText('Digit 1'), {
    clipboardData: { getData: () => '000000' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Verify phone' }));
  await screen.findByRole('heading', { name: 'Verify a card' });
}
describe('connected verification page', () => {
  it('checks email, sends/checks SMS and completes only from the final API status', async () => {
    setup();
    await verifyPhone();
    fireEvent.click(screen.getByRole('button', { name: 'Verify card' }));
    await screen.findByRole('heading', { name: 'Verification completed!' });
    expect(mocks.mutation).toHaveBeenCalledTimes(4);
    expect(mocks.mutation.mock.calls[2][1]).toEqual({
      phoneNumber: '+14155550100',
      code: '000000',
    });
  });
  it('shows a rate limit even when the server returns three passed checks', async () => {
    mocks.rateLimited = true;
    setup();
    await verifyPhone();
    fireEvent.click(screen.getByRole('button', { name: 'Verify card' }));
    await screen.findByRole('heading', {
      name: 'Too many verification attempts',
    });
    expect(
      screen.queryByRole('heading', { name: 'Verification completed!' }),
    ).toBeNull();
    mocks.rateLimited = false;
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    await screen.findByRole('heading', { name: 'Verification completed!' });
    expect(
      mocks.mutation.mock.calls.filter(
        ([document]) => document === AddVerificationCardDocument,
      ),
    ).toHaveLength(1);
  });
  it('never bootstraps email for an account mismatch', async () => {
    setup('other');
    await screen.findByRole('heading', {
      name: 'Check your company and profile',
    });
    expect(mocks.mutation).not.toHaveBeenCalled();
  });
  it('shows admin required before attempting any check', async () => {
    account.computeVerification!.canVerify = false;
    setup();
    await screen.findByRole('heading', { name: 'Ask an admin to verify' });
    await waitFor(() => expect(mocks.mutation).not.toHaveBeenCalled());
  });
});
