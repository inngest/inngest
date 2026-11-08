import { useEffect, useState } from 'react';

// Headless UI's `active` state includes pointer hover. Search inputs also match
// :focus-visible after a pointer click, so neither state identifies keyboard
// navigation. Listen before the options open, without handling or cancelling
// any selection/navigation events.
export function useMenuInputModality() {
  const [keyboardNavigation, setKeyboardNavigation] = useState(false);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (!['Shift', 'Control', 'Alt', 'Meta'].includes(event.key)) {
        setKeyboardNavigation(true);
      }
    };
    const onPointer = () => setKeyboardNavigation(false);

    document.addEventListener('keydown', onKeyDown, true);
    document.addEventListener('pointerdown', onPointer, true);
    document.addEventListener('pointermove', onPointer, true);
    return () => {
      document.removeEventListener('keydown', onKeyDown, true);
      document.removeEventListener('pointerdown', onPointer, true);
      document.removeEventListener('pointermove', onPointer, true);
    };
  }, []);

  return keyboardNavigation;
}
