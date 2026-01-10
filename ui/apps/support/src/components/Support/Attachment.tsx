import { useServerFn } from "@tanstack/react-start";
import { useQuery } from "@tanstack/react-query";
import { RiAttachment2 } from "@remixicon/react";
import type { Attachment } from "@/data/plain";
import { getAttachmentDownloadUrl } from "@/data/plain";

export function Attachment({ attachmentId }: { attachmentId: string }) {
  const attachmentDownloadUrlFn = useServerFn(getAttachmentDownloadUrl);
  const { data, isLoading } = useQuery({
    queryKey: ["attachmentDownloadUrl", attachmentId],
    queryFn: () => attachmentDownloadUrlFn({ data: { attachmentId } }),
  });

  if (isLoading) {
    return <div className="text-muted text-sm">Loading...</div>;
  }

  if (!data) return null;

  if (data.attachment.fileMimeType.match(/image\/.*/)) {
    return (
      <img
        src={data.downloadUrl}
        alt={data.attachment.fileName}
        title={data.attachment.fileName}
        className="max-w-sm rounded border border-subtle"
      />
    );
  }
  // Generic attachment
  return (
    <a
      href={data.downloadUrl}
      target="_blank"
      rel="noopener noreferrer"
      className="py-2 px-3 flex items-center gap-2 rounded border border-subtle text-muted text-sm hover:bg-canvasSubtle"
    >
      <RiAttachment2 className="h-4 w-4" />
      {data.attachment.fileName}
    </a>
  );
}
