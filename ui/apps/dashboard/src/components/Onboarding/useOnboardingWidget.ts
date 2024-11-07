import { useEffect, useState } from 'react';

import { getProdApps } from '@/components/Onboarding/actions';

const useOnboardingWidget = () => {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    const fetchProductionApps = async () => {
      try {
        const apps = await getProdApps();
        // Default to true only when user doesn't have prod apps
        setIsOpen(apps.length === 0);
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
