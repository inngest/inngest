import { useState } from 'react';

const useOnboardingWidget = () => {
  const [isOpen, setIsOpen] = useState(true);

  const showWidget = () => setIsOpen(true);
  const closeWidget = () => setIsOpen(false);

  return {
    isWidgetOpen: isOpen,
    showWidget,
    closeWidget,
  };
};

export default useOnboardingWidget;
