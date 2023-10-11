import { SignIn } from '@clerk/nextjs';

import SplitView from '@/app/(logged-out)/SplitView';

export default function SignInPage() {
  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <SignIn />
      </div>
    </SplitView>
  );
}
