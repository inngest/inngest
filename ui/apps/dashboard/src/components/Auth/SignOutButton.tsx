import { useClerk } from '@clerk/tanstack-react-start';
import { RiLogoutCircleLine } from '@remixicon/react';

export const SignOutButton = ({
  isMarketplace = false,
}: {
  isMarketplace?: boolean;
}) => {
  const { signOut, session } = useClerk();

  const content = (
    <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start h-full w-full p-2">
      <RiLogoutCircleLine className="text-muted mr-2 h-4 w-4" />
      <div>Sign Out </div>
    </div>
  );

  if (!isMarketplace) {
    // Sign out via Clerk.
    return (
      <button
        className="h-full w-full"
        onClick={async () => {
          await signOut({
            sessionId: session?.id,
            redirectUrl: '/sign-in/choose',
          });
        }}
      >
        {content}
      </button>
    );
  }

  // Sign out via our backend.
  return <a href={`${import.meta.env.VITE_API_URL}/v1/logout`}>{content}</a>;
};
