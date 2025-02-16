import { createContext } from 'react';
import type { useOrganization, useUser } from '@clerk/shared/react';

type PathCreator = {
  onboardingSteps: (p: { envSlug?: string; step?: string; ref?: string }) => any;
};

interface DIContextType {
  Link: React.ComponentType<React.AnchorHTMLAttributes<HTMLAnchorElement>>;
  pathCreator: PathCreator;
  useOrganization: typeof useOrganization;
  useUser: typeof useUser;
}

export const DIContext = createContext<DIContextType | null>(null);
