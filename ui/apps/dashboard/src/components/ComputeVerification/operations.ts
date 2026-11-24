import { graphql } from '@/gql';

export const ComputeVerificationDocument = graphql(
  `query ComputeVerification { account { id name computeVerification { status canVerify checks { kind passed display } } ents { sandboxes { enabled } } } }`,
);
export const CheckVerificationEmailDocument = graphql(
  `mutation CheckVerificationEmail { checkComputeVerificationEmail { error cardClientSecret verification { status canVerify checks { kind passed display } } } }`,
);
export const SendVerificationCodeDocument = graphql(
  `mutation SendVerificationCode($phoneNumber: String!) { sendComputeVerificationCode(phoneNumber: $phoneNumber) { error cardClientSecret verification { status canVerify checks { kind passed display } } } }`,
);
export const CheckVerificationCodeDocument = graphql(
  `mutation CheckVerificationCode($phoneNumber: String!, $code: String!) { checkComputeVerificationCode(phoneNumber: $phoneNumber, code: $code) { error cardClientSecret verification { status canVerify checks { kind passed display } } } }`,
);
export const AddVerificationCardDocument = graphql(
  `mutation AddVerificationCard($paymentMethodID: String!) { addComputeVerificationCard(paymentMethodID: $paymentMethodID) { error cardClientSecret verification { status canVerify checks { kind passed display } } } }`,
);
export const CheckVerificationCardDocument = graphql(
  `mutation CheckVerificationCard($setupIntentID: String!) { checkComputeVerificationCard(setupIntentID: $setupIntentID) { error cardClientSecret verification { status canVerify checks { kind passed display } } } }`,
);
