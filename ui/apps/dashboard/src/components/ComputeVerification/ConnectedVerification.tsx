import { useOrganizationList } from '@clerk/tanstack-react-start';
import type { ProfileDisplayType } from '@/queries/server/profile';
import { pathCreator } from '@/utils/urls';
import { VerificationPage } from './VerificationPage';
import { adaptVerification } from './adapter';
import { useComputeVerification } from './useComputeVerification';
import { StripeVerification } from './StripeVerification';
import type { EntrySource, VerificationActions } from './types';
export function ConnectedVerification({
  account_id,
  source,
  return_to,
  profile,
  profileID,
  onManageEmail,
}: {
  account_id: string;
  source: EntrySource;
  return_to?: string;
  profile: ProfileDisplayType;
  profileID: string;
  onManageEmail: () => void;
}): JSX.Element {
  const controller = useComputeVerification(account_id);
  const account = controller.account;
  // The link names an account UUID; show the user's matching organization.
  const { userMemberships } = useOrganizationList({
    userMemberships: { infinite: true },
  });
  const linkedAccountName = userMemberships.data?.find(
    (membership) =>
      (membership.organization.publicMetadata as { accountID?: string })
        .accountID === account_id,
  )?.organization.name;
  const state = adaptVerification(
    account?.computeVerification,
    {
      accountID: account?.id ?? account_id,
      companyName: account?.name ?? profile.orgName ?? 'Current account',
      profileID,
      displayName: profile.displayName,
      maskedEmail: '',
    },
    account_id,
    source,
    controller.transient,
  );
  if (state.originatingAccount)
    state.originatingAccount.companyName =
      linkedAccountName ?? 'the account from your link';
  if (controller.loading && !account) state.status = 'loading';
  else if (controller.readError) state.status = 'unavailable';
  state.pending = controller.pending;
  const baseActions: VerificationActions = {
    onSendCode: async (phoneNumber) => {
      await controller.run({ kind: 'send', phoneNumber });
    },
    onVerifyCode: async (code) => {
      if (controller.transient.phoneNumber)
        await controller.run({
          kind: 'code',
          phoneNumber: controller.transient.phoneNumber,
          code,
        });
    },
    onResendCode: async () => {
      if (controller.transient.phoneNumber)
        await controller.run({
          kind: 'send',
          phoneNumber: controller.transient.phoneNumber,
        });
    },
    onChangeNumber: controller.changeNumber,
    onContinueEmail: async () => {
      await controller.run({ kind: 'email' });
    },
    onManageEmail,
    onSubmitCard: () => {},
    onSwitchCompany: () => {},
    onSwitchProfile: () => {},
    onRetry: controller.retry,
    onReturn: () => {
      window.location.assign(return_to ?? '/env/production');
    },
    onContactSupport: () => {
      window.location.assign(pathCreator.support());
    },
  };
  return (
    <StripeVerification controller={controller}>
      {(adapter) => (
        <VerificationPage
          profile={profile}
          showOnboardingWidget={() => {
            window.location.assign('/env/production');
          }}
          state={state}
          actions={{ ...baseActions, onSubmitCard: adapter.onSubmitCard }}
          cardReady={adapter.cardReady}
          renderCardFields={adapter.renderCardFields}
        />
      )}
    </StripeVerification>
  );
}
