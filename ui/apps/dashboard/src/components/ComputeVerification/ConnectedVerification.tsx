import type { ProfileDisplayType } from '@/queries/server/profile';
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
      window.location.assign('mailto:support@inngest.com');
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
