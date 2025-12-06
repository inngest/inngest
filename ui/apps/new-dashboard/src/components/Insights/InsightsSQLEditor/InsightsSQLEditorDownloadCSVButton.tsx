import { Button } from "@inngest/components/Button/NewButton";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@inngest/components/Tooltip/Tooltip";

interface InsightsSQLEditorDownloadCSVButtonProps {
  temporarilyHide?: boolean;
}

export function InsightsSQLEditorDownloadCSVButton({
  temporarilyHide = false,
}: InsightsSQLEditorDownloadCSVButtonProps) {
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

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <Button
            appearance="ghost"
            disabled
            kind="secondary"
            label="Download as .csv"
            size="medium"
          />
        </span>
      </TooltipTrigger>
      <TooltipContent>Coming soon</TooltipContent>
    </Tooltip>
  );
}
