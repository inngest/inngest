import { RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';

import { Button } from '../Button';
import { TableBlankState } from '../Table/TableBlankState';
import { ExperimentsIcon } from '../icons/sections/Experiments';

export const EXPERIMENTS_DOCS_URL = 'https://www.inngest.com/docs/features/step-experimentation';

type Props = {
  title: React.ReactNode;
  description: React.ReactNode;
  onRefresh: () => void;
};

export function ExperimentsBlankState({ title, description, onRefresh }: Props) {
  return (
    <TableBlankState
      icon={<ExperimentsIcon />}
      title={title}
      description={description}
      actions={
        <>
          <Button
            kind="primary"
            appearance="outlined"
            label="Refresh"
            icon={<RiRefreshLine />}
            iconSide="left"
            onClick={() => onRefresh()}
          />
          <Button
            kind="primary"
            appearance="solid"
            label="Go to docs"
            href={EXPERIMENTS_DOCS_URL}
            target="_blank"
            icon={<RiExternalLinkLine />}
            iconSide="left"
          />
        </>
      }
    />
  );
}
