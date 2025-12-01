import { useClerk } from "@clerk/tanstack-react-start";
import { RiLogoutCircleLine } from "@remixicon/react";

export const SignOutButton = ({
  isMarketplace = false,
}: {
  isMarketplace?: boolean;
}) => {
  const { signOut, session } = useClerk();

  const content = (
    <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
      <RiLogoutCircleLine className="text-muted mr-2 h-4 w-4" />
      <div>Sign Out </div>
    </div>
  );

  if (!isMarketplace) {
    // Sign out via Clerk.
    return (
      <button
        onClick={async () => {
          await signOut({
            sessionId: session?.id,
            redirectUrl: "/sign-in/choose",
          });
        }}
      >
        {content}
      </button>
    );
  }

  // Sign out via our backend.
  return (
    // @ts-expect-error - TANSTACK TODO: fix when routes land
    <Link to={`${import.meta.env.VITE_API_URL}/v1/logout`}>{content}</Link>
  );
};
