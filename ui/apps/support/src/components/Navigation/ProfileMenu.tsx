import {
  Listbox,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
} from "@headlessui/react";
import { RiLogoutCircleLine } from "@remixicon/react";
import { useClerk } from "@clerk/tanstack-react-start";
import { Link } from "@tanstack/react-router";
import { Button } from "@inngest/components/Button/NewButton";

type Props = React.PropsWithChildren<{
  isAuthenticated: boolean;
}>;

export const ProfileMenu = ({ children, isAuthenticated }: Props) => {
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
        <ListboxOptions className="bg-canvasBase border-muted shadow-primary absolute -right-48 bottom-4 z-50 ml-8 w-[199px] rounded border ring-0 focus:outline-none">
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
