import { UserProfile } from '@clerk/nextjs';

const UserProfilePage = () => (
  <div className="min-h-0 flex-1">
    <UserProfile
      path="/settings/account"
      routing="path"
      appearance={{
        variables: {
          colorAlphaShade: '#FFF', // white to hide the Clerk's scrollbar
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

export default UserProfilePage;
