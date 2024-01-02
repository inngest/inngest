'use client';

import { useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { Button } from '@inngest/components/Button';

import DeployModal from './DeployModal';

export default function DeployButton() {
  const searchParams = useSearchParams();
  const hasDeployIntent = searchParams.get('intent') === 'deploy-modal';
  const [isDeployModalVisible, setIsDeployModalVisible] = useState<boolean>(hasDeployIntent);

  return (
    <>
      <Button kind="primary" btnAction={() => setIsDeployModalVisible(true)} label="Deploy" />
      <DeployModal isOpen={isDeployModalVisible} onClose={() => setIsDeployModalVisible(false)} />
    </>
  );
}
