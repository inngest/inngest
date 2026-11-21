import { createContext, useContext, useState, type ReactNode } from 'react';
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
  useNavigate,
} from '@tanstack/react-router';
import { Button } from '@inngest/components/Button';
import { ThemeProvider } from 'next-themes';
import { VerificationLayout } from '../ComputeVerification';
import type { VerificationActions, VerificationState } from '../types';
import { DemoFooter, DemoHeader } from './DemoShell';
import { company, personal } from './scenarios';

export const demoCompanies = [
  { accountID: company.accountID, companyName: company.companyName },
  { accountID: personal.accountID, companyName: personal.companyName },
  { accountID: 'acme', companyName: 'Acme Research' },
];
export const demoProfiles = [
  {
    profileID: company.profileID,
    displayName: company.displayName,
    maskedEmail: company.maskedEmail,
  },
  {
    profileID: personal.profileID,
    displayName: 'Sanjana (personal)',
    maskedEmail: personal.maskedEmail,
  },
  { profileID: 'alex', displayName: 'Alex', maskedEmail: 'a••••@acme.example' },
];

type DemoContextValue = {
  state: VerificationState;
  actions: VerificationActions;
  content: ReactNode;
  onSelectCompany: (company: (typeof demoCompanies)[number]) => void;
  onSelectProfile: (profile: (typeof demoProfiles)[number]) => void;
};

const demoContext = createContext<DemoContextValue | null>(null);
function useDemo() {
  const context = useContext(demoContext);
  if (!context) throw new Error('Missing demo context');
  return context;
}
const rootRoute = createRootRoute({
  component: DemoLayout,
});
const verificationRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DemoVerificationContent,
});
const companiesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/organization-list',
  component: CompanyChooser,
});
const profilesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/sign-in/choose',
  component: ProfileChooser,
});
const otherRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '$',
  component: AppDestination,
});
const routeTree = rootRoute.addChildren([
  verificationRoute,
  companiesRoute,
  profilesRoute,
  otherRoute,
]);

export function DemoNavigation({
  children,
  ...context
}: Omit<DemoContextValue, 'content'> & { children: ReactNode }) {
  const [router] = useState(() =>
    createRouter({
      routeTree,
      history: createMemoryHistory({ initialEntries: ['/'] }),
    }),
  );
  return (
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem={false}>
      <demoContext.Provider value={{ ...context, content: children }}>
        <RouterProvider router={router} />
      </demoContext.Provider>
    </ThemeProvider>
  );
}

function DemoVerificationContent() {
  return useDemo().content;
}

function DemoLayout() {
  const { state, actions } = useDemo();
  return (
    <VerificationLayout
      header={<DemoHeader state={state} />}
      footer={<DemoFooter onContactSupport={actions.onContactSupport} />}
    >
      <Outlet />
    </VerificationLayout>
  );
}

function BackToVerification() {
  const navigate = useNavigate();
  return (
    <Button
      kind="secondary"
      appearance="outlined"
      label="Back to verification"
      onClick={() => void navigate({ to: '/' })}
    />
  );
}

function CompanyChooser() {
  const { onSelectCompany } = useDemo();
  const navigate = useNavigate();
  return (
    <section className="flex w-full max-w-md flex-col gap-4">
      <h1 className="text-2xl font-medium">Choose a company</h1>
      <p className="text-muted text-sm">
        Demo destination for the app’s existing organization picker. Select a
        fixture explicitly to return to verification.
      </p>
      {demoCompanies.map((company) => (
        <Button
          key={company.accountID}
          kind="secondary"
          appearance="outlined"
          label={company.companyName}
          onClick={() => {
            onSelectCompany(company);
            void navigate({ to: '/' });
          }}
        />
      ))}
      <BackToVerification />
    </section>
  );
}

function ProfileChooser() {
  const { onSelectProfile } = useDemo();
  const navigate = useNavigate();
  return (
    <section className="flex w-full max-w-md flex-col gap-4">
      <h1 className="text-2xl font-medium">Choose a profile</h1>
      <p className="text-muted text-sm">
        Demo destination for the app’s existing account picker. Select a fixture
        explicitly to return to verification.
      </p>
      {demoProfiles.map((profile) => (
        <Button
          key={profile.profileID}
          kind="secondary"
          appearance="outlined"
          label={`${profile.displayName} · ${profile.maskedEmail}`}
          onClick={() => {
            onSelectProfile(profile);
            void navigate({ to: '/' });
          }}
        />
      ))}
      <BackToVerification />
    </section>
  );
}

function AppDestination() {
  return (
    <section className="flex w-full max-w-md flex-col gap-4">
      <h1 className="text-2xl font-medium">App navigation preview</h1>
      <p className="text-muted text-sm">
        The existing menu navigated to another app page. This standalone demo
        keeps that destination local; it does not change your company or
        profile.
      </p>
      <BackToVerification />
    </section>
  );
}
