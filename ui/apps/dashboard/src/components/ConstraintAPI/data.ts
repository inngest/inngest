import { graphql } from '@/gql';

// Query to check enrollment status (any state)
export const ConstraintAPIEnrollmentQuery = graphql(`
  query ConstraintAPIEnrollment {
    account {
      id
      constraintAPIEnrolled
    }
  }
`);

// Query to check if API is active/in effect
export const ConstraintAPIInEffectQuery = graphql(`
  query ConstraintAPIInEffect {
    account {
      id
      constraintAPIEnrolled(inEffect: true)
    }
  }
`);

// Mutation for enrollment
export const EnrollToConstraintAPIMutation = graphql(`
  mutation EnrollToConstraintAPI {
    enrollToConstraintAPI
  }
`);

export type ConstraintAPIData = {
  isEnrolled: boolean; // Has the user enrolled?
  isInEffect: boolean; // Is the API currently active?
  displayState: 'not_enrolled' | 'pending' | 'active';
};

export function parseConstraintAPIData(
  enrolledData: any,
  inEffectData: any,
): ConstraintAPIData | null {
  if (!enrolledData?.account || !inEffectData?.account) {
    return null;
  }

  const isEnrolled = enrolledData.account.constraintAPIEnrolled ?? false;
  const isInEffect = inEffectData.account.constraintAPIEnrolled ?? false;

  let displayState: ConstraintAPIData['displayState'];
  if (!isEnrolled) {
    displayState = 'not_enrolled';
  } else if (isEnrolled && !isInEffect) {
    displayState = 'pending';
  } else {
    displayState = 'active';
  }

  return { isEnrolled, isInEffect, displayState };
}
