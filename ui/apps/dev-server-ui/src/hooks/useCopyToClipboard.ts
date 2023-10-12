import { useState } from 'react';

export default function useCopyToClipboard() {
  const [clickedState, setClickedState] = useState(false);

  const handleCopyClick = async (code: string) => {
    setClickedState(true);

    try {
      await navigator.clipboard.writeText(code);
    } catch (error) {
      console.error('Failed to copy:', error);
    }

    setTimeout(() => {
      setClickedState(false);
    }, 1000);
  };

  return {
    handleCopyClick,
    isCopying: clickedState,
  };
}
