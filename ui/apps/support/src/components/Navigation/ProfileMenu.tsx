import {
  Listbox,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
} from "@headlessui/react";
import { RiLogoutCircleLine } from "@remixicon/react";
import { useClerk } from "@clerk/tanstack-react-start";
import { Link } from "@tanstack/react-router";
import { Button } from "@inngest/components/Button/Button";
import { cn } from "@inngest/components/utils/classNames";

type Props = React.PropsWithChildren<{
  isAuthenticated: boolean;
  email?: string;
  organizationName?: string;
  position?: "above" | "below";
}>;

export const ProfileMenu = ({
  children,
  isAuthenticated,
  email,
  organizationName,
  position = "above",
}: Props) => {
  if (!isAuthenticated) {
    return (
      <Link to="/sign-in/$">
        <Button appearance="outlined" label="Sign In" />
      </Link>
    );
  }

  return (
    <Listbox>
      <ListboxButton className="w-full cursor-pointer ring-0">
        {children}
      </ListboxButton>
      <div className="relative">
        <ListboxOptions
          className={cn(
            "bg-canvasBase border-muted shadow-primary absolute z-50 ml-2 w-[199px] rounded border ring-0 focus:outline-none",
            position === "above" ? "left-0 -bottom-4" : "right-0 top-6",
          )}
        >
          <div className="text-muted m-2 flex flex-col min-h-8 cursor-default items-start gap-1 px-2 text-[13px]">
            <div className="font-medium">{email}</div>
            <div className="text-muted text-xs">{organizationName}</div>
          </div>
          <ListboxOption
            className="text-muted hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="signOut"
          >
            <SignOut />
          </ListboxOption>
        </ListboxOptions>
      </div>
    </Listbox>
  );
};

function SignOut() {
  const { signOut, session } = useClerk();

  const content = (
    <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
      <RiLogoutCircleLine className="text-muted mr-2 h-4 w-4" />
      <div>Sign Out</div>
    </div>
  );

  return (
    <button
      onClick={async () => {
        await signOut({
          sessionId: session?.id,
          redirectUrl: "/sign-in/$",
        });
      }}
    >
      {content}
    </button>
  );
}
