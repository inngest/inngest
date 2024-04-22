import { useState } from 'react';
import { toast } from 'sonner';

export function useCopyToClipboard() {
  const [clickedState, setClickedState] = useState(false);

  const handleCopyClick = async (code: string) => {
    setClickedState(true);

    try {
      await navigator.clipboard.writeText(code);
      toast.success(`Copied to clipboard`);
    } catch (error) {
      console.error('Failed to copy:', error);
      toast.error(`Failed to copy`);
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
