import { useEffect, useState } from 'react';

import { getProdApps } from '@/components/Onboarding/actions';

const useOnboardingWidget = () => {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    const fetchProductionApps = async () => {
      try {
        const savedPreference = localStorage.getItem('showOnboardingWidget');
        if (savedPreference !== null) {
          // If a preference is saved, use it
          setIsOpen(JSON.parse(savedPreference));
          return;
        }
        const result = await getProdApps();
        const hasAppsOrUnattachedSyncs = result
          ? result.apps.length > 0 || result.unattachedSyncs.length > 0
          : // In case of data fetching error, we don't wanna fail the page
            true;
        // Show widget by default when user doesn't have prod apps
        const defaultState = !hasAppsOrUnattachedSyncs;
        setIsOpen(defaultState);
        localStorage.setItem('showOnboardingWidget', JSON.stringify(defaultState));
      } catch (error) {
        console.error('Error in useOnboardingWidget:', error);
      }
    };

    fetchProductionApps();
  }, []);

  const showWidget = () => {
    setIsOpen(true);
    localStorage.setItem('showOnboardingWidget', JSON.stringify(true));
  };
  const closeWidget = () => {
    setIsOpen(false);
    localStorage.setItem('showOnboardingWidget', JSON.stringify(false));
  };

  return {
    isWidgetOpen: isOpen,
    showWidget,
    closeWidget,
  };
};

export default useOnboardingWidget;
