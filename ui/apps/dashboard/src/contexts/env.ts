import { createContext } from 'react';

const defaultValue = {
  id: '',
  name: '',
  slug: '',
};

export const EnvContext = createContext(defaultValue);
