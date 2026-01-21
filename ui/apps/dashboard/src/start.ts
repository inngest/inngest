import { clerkMiddleware } from '@clerk/tanstack-react-start/server';
import { createStart } from '@tanstack/react-start';
import { securityMiddleware } from './middleware/securityMiddleware';

export const startInstance = createStart(() => {
  return {
    requestMiddleware: [clerkMiddleware(), securityMiddleware],
  };
});
