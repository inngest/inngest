'use client';

import { Button } from '@inngest/components/Button/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiCodeBlock, RiDownloadLine, RiTableView } from '@remixicon/react';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { useDownloadInsightsResults } from './hooks/useDownloadInsightsResults';

interface InsightsSQLEditorDownloadCSVButtonProps {
  temporarilyHide?: boolean;
}

export function InsightsSQLEditorDownloadCSVButton({
  temporarilyHide = false,
}: InsightsSQLEditorDownloadCSVButtonProps) {
  const { data, status, queryName } = useInsightsStateMachineContext();
  const { downloadAsCSV, downloadAsJSON } = useDownloadInsightsResults(data, queryName);

  // Maintain layout consistency when the button is temporarily hidden.
  if (temporarilyHide) {
    return (
      <Button
        appearance="ghost"
        className="invisible"
        disabled
        kind="secondary"
        label=""
        size="medium"
      />
    );
  }

  const hasData = status === 'success' && data && data.rows.length > 0;

  if (!hasData) {
    return (
      <Button
        appearance="outlined"
        disabled
        kind="secondary"
        label="Download"
        size="medium"
        icon={<RiDownloadLine className="h-4 w-4" />}
        iconSide="left"
      />
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          appearance="outlined"
          kind="secondary"
          label="Download"
          size="medium"
          icon={<RiDownloadLine className="h-4 w-4" />}
          iconSide="left"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={downloadAsCSV}>
          <RiTableView className="h-4 w-4" />
          Download as .csv
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={downloadAsJSON}>
          <RiCodeBlock className="h-4 w-4" />
          Download as .json
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
