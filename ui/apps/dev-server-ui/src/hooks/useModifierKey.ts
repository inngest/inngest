import { useEffect, useState } from 'react';

const useModifierKey = () => {
  const [modifierKey, setModifierKey] = useState('');
  useEffect(() => {
    setModifierKey(/(Mac|iPhone|iPod|iPad)/i.test(navigator.platform) ? '⌘' : 'Ctrl');
  }, []);

  return modifierKey;
};

export default useModifierKey;
