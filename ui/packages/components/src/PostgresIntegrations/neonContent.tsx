import { IconNeon } from '../icons/platforms/Neon';
import { type ConnectPostgresIntegrationContent } from './types';

export const neonConnectContent: ConnectPostgresIntegrationContent = {
  title: 'Neon',
  logo: <IconNeon className="text-onContrast" size={20} />,
  description:
    'This integration enables you to host your Inngest functions on the Vercel platform and automatically sync them every time you deploy code.',
  url: '',
  step: {
    1: {
      title: 'Authorize',
      description: 'Add your postgres credentials so Inngest can access your database.',
    },
    2: {
      title: 'Format WAL',
      description: 'Change/confirm your Write-Ahead Logging (WAL) is set to Logical.',
    },
    3: {
      title: 'Connect Neon to Inngest',
      description: 'Add your postgres credentials so Inngest can access your database.',
    },
  },
};