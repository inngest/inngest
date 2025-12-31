import { useState } from 'react';
import { useOrganization, useUser } from '@clerk/tanstack-react-start';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Textarea } from '@inngest/components/Forms/Textarea';
import { Modal } from '@inngest/components/Modal/Modal';
import { Select } from '@inngest/components/Select/Select';
import { capitalCase } from 'change-case';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { type CheckoutItem } from './CheckoutModal';

type ConfirmPlanChangeModalProps = {
  action: 'upgrade' | 'downgrade' | 'cancel';
  items: CheckoutItem[];
  onCancel: () => void;
  onSuccess: () => void;
};

const UpdatePlanDocument = graphql(`
  mutation UpdatePlan($planSlug: String!) {
    updatePlan(slug: $planSlug) {
      plan {
        id
        name
      }
    }
  }
`);

const SubmitChurnSurveyDocument = graphql(`
  mutation SubmitChurnSurvey(
    $reason: String!
    $feedback: String
    $email: String!
    $accountID: UUID!
    $clerkUserID: String!
  ) {
    submitChurnSurvey(
      reason: $reason
      feedback: $feedback
      email: $email
      accountID: $accountID
      clerkUserID: $clerkUserID
    )
  }
`);

// Note - this is currently only used for downgrades as Stripe's Payment Intents
// does not allow for saving a default payment method by default when creating
// a subscription. This is also OK as it ensure that we always have a correct
// and updated Credit Card on file when creating a new subscription.
const CHURN_REASONS = [
  { id: 'too-expensive', name: 'Too expensive' },
  { id: 'missing-features', name: 'Missing features' },
  { id: 'too-complex', name: 'Too complex to use' },
  { id: 'found-alternative', name: 'Found a better alternative' },
  { id: 'no-longer-needed', name: 'No longer needed' },
  { id: 'poor-performance', name: 'Poor performance / Issues' },
  { id: 'other', name: 'Other' },
];

export default function ConfirmPlanChangeModal({
  action,
  items,
  onCancel,
  onSuccess,
}: ConfirmPlanChangeModalProps) {
  const [uiError, setUiError] = useState('');
  const [showSurvey, setShowSurvey] = useState(false);
  const [churnReason, setChurnReason] = useState<{
    id: string;
    name: string;
  } | null>(null);
  const [churnFeedback, setChurnFeedback] = useState('');
  const [{ error: apiError }, updatePlan] = useMutation(UpdatePlanDocument);
  const [{ error: churnError }, submitChurnSurvey] = useMutation(
    SubmitChurnSurveyDocument,
  );
  const { user } = useUser();
  const { organization } = useOrganization();
  const error = apiError?.message || churnError?.message || uiError;

  const handlePlanChange = async () => {
    const isCancellation = action === 'cancel';
    const isDowngrade = action === 'downgrade';
    const shouldShowSurvey = isCancellation || isDowngrade;

    // For cancellations and downgrades, show survey first
    if (shouldShowSurvey && !showSurvey) {
      setShowSurvey(true);
      return;
    }

    // NOTE - We only support one Stripe plan/product in the API currently
    // so we just grab the first item
    const planSlug = items[0]?.planSlug;
    if (!planSlug) {
      return setUiError(
        'Unable to change your plan - Invalid Plan Slug. Please contact support.',
      );
    }

    // Submit churn survey if cancelling/downgrading and survey data exists
    if (shouldShowSurvey && churnReason && user && organization) {
      const accountId = organization.publicMetadata.accountID;
      const email = user.primaryEmailAddress?.emailAddress;

      if (typeof accountId === 'string' && email) {
        try {
          await submitChurnSurvey({
            reason: churnReason.id,
            feedback: churnFeedback || null,
            email,
            accountID: accountId,
            clerkUserID: user.id,
          });
        } catch (error) {
          console.error('Failed to submit churn survey:', error);
          // Continue with cancellation even if survey fails
        }
      }
    }

    await updatePlan({ planSlug });
    onSuccess();
  };

  const handleSurveySubmit = () => {
    if (!churnReason) {
      setUiError('Please select a reason for canceling.');
      return;
    }
    setUiError('');
    handlePlanChange();
  };

  const amount = items.reduce((total, item) => {
    return total + item.amount * item.quantity;
  }, 0);
  const planName = items.map((item) => item.name).join(', ');
  const isCancellation = action === 'cancel';
  const isDowngrade = action === 'downgrade';
  const shouldShowSurvey = isCancellation || isDowngrade;

  return (
    <Modal
      className="flex min-w-[600px] max-w-xl flex-col gap-4"
      isOpen={true}
      onClose={onCancel}
    >
      <Modal.Header>
        {isCancellation ? (
          <>Cancel Your Subscription</>
        ) : isDowngrade ? (
          <>Downgrade to {planName}</>
        ) : (
          <>
            {capitalCase(action)} to {planName}
          </>
        )}
      </Modal.Header>

      <Modal.Body>
        {shouldShowSurvey ? (
          showSurvey ? (
            <>
              <p className="mb-4">
                {isCancellation
                  ? "We're sorry to see you go! Help us improve by telling us why you're canceling."
                  : "We'd love to understand why you're downgrading to help us improve our service."}
              </p>
              <div className="space-y-4">
                <div className="w-full">
                  <label className="mb-2 block text-sm font-medium">
                    {isCancellation
                      ? 'Primary reason for canceling'
                      : 'Primary reason for downgrading'}{' '}
                    <span className="text-error">*</span>
                  </label>
                  <div className="w-full">
                    <Select
                      value={churnReason}
                      onChange={setChurnReason}
                      isLabelVisible={false}
                      className="bg-canvasBase w-full"
                    >
                      <Select.Button className="bg-canvasBase focus:outline-primary-moderate w-full outline-2 transition-all focus:outline focus:ring-0">
                        {churnReason ? churnReason.name : 'Select a reason...'}
                      </Select.Button>
                      <Select.Options className="w-full">
                        {CHURN_REASONS.map((reason) => (
                          <Select.Option key={reason.id} option={reason}>
                            {reason.name}
                          </Select.Option>
                        ))}
                      </Select.Options>
                    </Select>
                  </div>
                </div>
                <div>
                  <label className="mb-2 block text-sm font-medium">
                    Additional feedback (optional)
                  </label>
                  <Textarea
                    value={churnFeedback}
                    onChange={setChurnFeedback}
                    placeholder="Tell us more about your experience..."
                    rows={3}
                  />
                </div>
              </div>
              <Alert severity="warning" className="mt-6">
                <b>Note:</b> After submitting, your{' '}
                {isCancellation ? 'cancellation' : 'downgrade'} will be
                processed immediately and you will lose access to your current
                plan features.
              </Alert>
            </>
          ) : (
            <>
              <p className="mb-2">
                <b>This is an immediate action.</b> Please confirm before{' '}
                {isCancellation ? 'canceling' : 'downgrading'} your plan.{' '}
              </p>
              <ul className="list-inside list-disc">
                <li>
                  Once {isCancellation ? 'canceled' : 'downgraded'}, you will
                  lose access to your current plan and its features{' '}
                  <b>immediately</b>.
                </li>
                <li>
                  You will be credited for any unused time from your current
                  plan, calculated on a prorated basis.
                </li>
              </ul>
            </>
          )
        ) : (
          <p>You have chosen wisely - Please confirm your upgrade!</p>
        )}
        {!shouldShowSurvey || !showSurvey ? (
          <p className="my-4 font-semibold">
            New monthly cost: ${amount / 100}
          </p>
        ) : null}
        <div className="mt-6 flex flex-row justify-end gap-3">
          {shouldShowSurvey && showSurvey ? (
            <>
              <Button
                kind="secondary"
                onClick={() => setShowSurvey(false)}
                label="Back"
              />
              <Button
                kind="danger"
                onClick={handleSurveySubmit}
                label={
                  isCancellation
                    ? 'Submit & Cancel Plan'
                    : 'Submit & Downgrade Plan'
                }
              />
            </>
          ) : (
            <Button
              kind={shouldShowSurvey ? 'danger' : 'primary'}
              onClick={handlePlanChange}
              label={
                shouldShowSurvey ? 'Continue' : `Confirm ${capitalCase(action)}`
              }
            />
          )}
        </div>
        {Boolean(error) && <Alert severity="error">{error}</Alert>}
      </Modal.Body>
    </Modal>
  );
}
