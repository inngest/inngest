import { useEffect, useState } from 'react';

import { getProdApps } from '@/components/Onboarding/actions';

const useOnboardingWidget = () => {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    const fetchProductionApps = async () => {
      try {
        const { apps, unattachedSyncs } = await getProdApps();
        const hasAppsOrUnattachedSyncs = apps.length > 0 || unattachedSyncs.length > 0;
        // Show widget by default when user doesn't have prod apps
        setIsOpen(!hasAppsOrUnattachedSyncs);
      } catch (error) {
        console.error('Error in useOnboardingWidget:', error);
      }
    };

    fetchProductionApps();
  }, []);

  const showWidget = () => setIsOpen(true);
  const closeWidget = () => setIsOpen(false);

  return {
    isWidgetOpen: isOpen,
    showWidget,
    closeWidget,
  };
};

export default useOnboardingWidget;
