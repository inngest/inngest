import { UserProfile } from '@clerk/nextjs';

export default function UserSettingsPage() {
  return (
    <div className="min-h-0 flex-1">
      <UserProfile
        routing="path"
        path="/settings/user"
        appearance={{
          elements: {
            rootBox: 'h-full',
            card: 'divide-x divide-slate-100',
            navbar: 'p-8 border-none',
            scrollBox: 'bg-white',
            pageScrollBox: '[scrollbar-width:none]', // hides the Clerk's scrollbar
            profileSectionTitle__danger: 'hidden',
            profileSectionContent__danger: 'hidden',
          },
        }}
      />
    </div>
  );
}
