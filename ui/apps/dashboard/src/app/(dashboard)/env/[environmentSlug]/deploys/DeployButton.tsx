'use client';

import { useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { Button } from '@inngest/components/Button';

import DeployModal from './DeployModal';

type DeployButtonProps = {
  environmentSlug: string;
};

export default function DeployButton({ environmentSlug }: DeployButtonProps) {
  const searchParams = useSearchParams();
  const hasDeployIntent = searchParams.get('intent') === 'deploy-modal';
  const [isDeployModalVisible, setIsDeployModalVisible] = useState<boolean>(hasDeployIntent);

  return (
    <>
      <Button kind="primary" btnAction={() => setIsDeployModalVisible(true)} label="Deploy" />
      <DeployModal
        isOpen={isDeployModalVisible}
        environmentSlug={environmentSlug}
        onClose={() => setIsDeployModalVisible(false)}
      />
    </>
  );
}
