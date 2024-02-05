import { UserProfile } from '@clerk/nextjs';
import colors from 'tailwindcss/colors';

export default function UserAccountSettingsPage() {
  return (
    <div className="min-h-0 flex-1">
      <UserProfile
        routing="path"
        path="/settings/account"
        appearance={{
          variables: {
            colorAlphaShade: colors.white, // white to hide the Clerk's scrollbar
          },
          elements: {
            rootBox: 'h-full',
            card: 'divide-x divide-slate-100',
            navbar: 'p-8 border-none',
            scrollBox: 'bg-white',
            pageScrollBox: '[scrollbar-width:none]', // hides the Clerk's scrollbar in Firefox
          },
        }}
      />
    </div>
  );
}
