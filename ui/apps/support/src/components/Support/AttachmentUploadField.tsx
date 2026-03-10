import { useRef, useState } from "react";
import { useServerFn } from "@tanstack/react-start";
import {
  RiAttachmentLine,
  RiCloseLine,
  RiFileLine,
  RiImageLine,
  RiLoader4Line,
} from "@remixicon/react";
import { Button } from "@inngest/components/Button";
import {
  ACCEPTED_FILE_TYPES,
  type AttachmentUploadContext,
  MAX_ATTACHMENTS,
  MAX_ATTACHMENT_SIZE_BYTES,
  getUploadUrl,
  validateAttachment,
} from "@/data/plain";

export type PendingAttachment = {
  id: string;
  file: File;
  status: "pending" | "uploading" | "uploaded" | "error";
  attachmentId?: string;
  error?: string;
};

type UseAttachmentUploadOptions = {
  userEmail?: string;
  context?: AttachmentUploadContext;
  onError?: (message: string | null) => void;
};

export function useAttachmentUpload({
  userEmail,
  context = "chat",
  onError,
}: UseAttachmentUploadOptions) {
  const getUploadUrlFn = useServerFn(getUploadUrl);
  const [attachments, setAttachments] = useState<PendingAttachment[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const isUploading = attachments.some((a) => a.status === "uploading");
  const uploadedAttachmentIds = attachments
    .filter((a) => a.status === "uploaded" && a.attachmentId)
    .map((a) => a.attachmentId!);

  async function uploadFile(file: File, clientId: string) {
    setAttachments((prev) =>
      prev.map((a) =>
        a.id === clientId ? { ...a, status: "uploading" as const } : a,
      ),
    );

    try {
      const urlResult = await getUploadUrlFn({
        data: {
          userEmail: userEmail || "",
          fileName: file.name,
          fileSizeBytes: file.size,
          context,
        },
      });

      if (
        !urlResult.success ||
        !urlResult.uploadFormUrl ||
        !urlResult.uploadFormData ||
        !urlResult.attachmentId
      ) {
        throw new Error(urlResult.error || "Failed to get upload URL");
      }

      const formData = new FormData();
      for (const field of urlResult.uploadFormData) {
        formData.append(field.key, field.value);
      }
      formData.append("file", file);

      const uploadRes = await fetch(urlResult.uploadFormUrl, {
        method: "POST",
        body: formData,
      });

      if (!uploadRes.ok) {
        throw new Error(`Upload failed with status ${uploadRes.status}`);
      }

      setAttachments((prev) =>
        prev.map((a) =>
          a.id === clientId
            ? {
                ...a,
                status: "uploaded" as const,
                attachmentId: urlResult.attachmentId,
              }
            : a,
        ),
      );
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Attachment upload failed";
      console.error("Error uploading attachment:", error);
      setAttachments((prev) =>
        prev.map((a) =>
          a.id === clientId
            ? {
                ...a,
                status: "error" as const,
                error: message,
              }
            : a,
        ),
      );
      onError?.(message);
    }
  }

  async function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const fileList = e.target.files;
    if (!fileList || fileList.length === 0) return;

    const selectedFiles = Array.from(fileList);
    e.target.value = "";

    setAttachments((prev) => {
      const remainingSlots = MAX_ATTACHMENTS - prev.length;
      if (remainingSlots <= 0) {
        onError?.(`You can attach a maximum of ${MAX_ATTACHMENTS} files.`);
        return prev;
      }

      const filesToProcess = selectedFiles.slice(0, remainingSlots);
      if (selectedFiles.length > remainingSlots) {
        onError?.(
          `Only ${remainingSlots} more file(s) can be added (max ${MAX_ATTACHMENTS}).`,
        );
      }

      const validEntries: PendingAttachment[] = [];
      for (const file of filesToProcess) {
        const validationError = validateAttachment(
          file.name,
          file.type,
          file.size,
        );
        if (validationError) {
          onError?.(validationError);
          continue;
        }
        const clientId = `${Date.now()}-${Math.random()
          .toString(36)
          .slice(2, 9)}`;
        validEntries.push({
          id: clientId,
          file,
          status: "pending",
        });
      }

      if (validEntries.length === 0) return prev;

      onError?.(null);

      // Kick off uploads outside the state setter
      setTimeout(() => {
        for (const entry of validEntries) {
          uploadFile(entry.file, entry.id);
        }
      }, 0);

      return [...prev, ...validEntries];
    });
  }

  function removeAttachment(id: string) {
    setAttachments((prev) => prev.filter((a) => a.id !== id));
    onError?.(null);
  }

  function openFilePicker() {
    fileInputRef.current?.click();
  }

  function clearAttachments() {
    setAttachments([]);
  }

  return {
    attachments,
    isUploading,
    uploadedAttachmentIds,
    fileInputRef,
    handleFileSelect,
    removeAttachment,
    openFilePicker,
    clearAttachments,
  };
}

type AttachmentUploadFieldProps = {
  attachments: PendingAttachment[];
  isUploading: boolean;
  isSubmitting: boolean;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  onFileSelect: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onRemoveAttachment: (id: string) => void;
  onAddClick: () => void;
  variant?: "default" | "compact";
  showHelpText?: boolean;
};

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function AttachmentUploadField({
  attachments,
  isUploading,
  isSubmitting,
  fileInputRef,
  onFileSelect,
  onRemoveAttachment,
  onAddClick,
  variant = "default",
  showHelpText = true,
}: AttachmentUploadFieldProps) {
  const isDisabled = isSubmitting || isUploading;
  const atLimit = attachments.length >= MAX_ATTACHMENTS;

  return (
    <div className="border-subtle bg-canvasBase flex flex-col gap-2 rounded-lg border px-3 py-2">
      {attachments.length > 0 && (
        <div className="flex flex-col gap-1.5">
          {attachments.map((attachment) => (
            <div
              key={attachment.id}
              className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-sm ${
                attachment.status === "error"
                  ? "border-red-300 bg-red-50 text-red-700"
                  : attachment.status === "uploading"
                  ? "border-subtle bg-canvasSubtle text-muted"
                  : attachment.status === "uploaded"
                  ? "border-green-200 bg-green-50 text-green-800"
                  : "border-subtle bg-canvasSubtle text-basis"
              }`}
            >
              {attachment.status === "uploading" ? (
                <RiLoader4Line className="h-4 w-4 animate-spin flex-shrink-0" />
              ) : attachment.file.type.startsWith("image/") ? (
                <RiImageLine className="h-4 w-4 flex-shrink-0" />
              ) : (
                <RiFileLine className="h-4 w-4 flex-shrink-0" />
              )}

              <span className="min-w-0 flex-1 truncate font-medium">
                {attachment.file.name}
              </span>

              <span className="flex-shrink-0 text-xs opacity-70">
                {formatFileSize(attachment.file.size)}
              </span>

              {attachment.status === "uploaded" && (
                <span className="flex-shrink-0 text-xs font-medium text-green-600">
                  Ready
                </span>
              )}
              {attachment.status === "uploading" && (
                <span className="flex-shrink-0 text-xs">Uploading...</span>
              )}
              {attachment.status === "error" && (
                <span
                  className="flex-shrink-0 text-xs font-medium text-red-600"
                  title={attachment.error}
                >
                  Failed
                </span>
              )}

              <button
                type="button"
                onClick={() => onRemoveAttachment(attachment.id)}
                className="text-muted hover:text-basis flex-shrink-0 rounded-full p-0.5 transition-colors hover:bg-canvasMuted"
                title="Remove"
                disabled={isSubmitting}
              >
                <RiCloseLine className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}

      <div>
        <input
          ref={fileInputRef as React.RefObject<HTMLInputElement>}
          type="file"
          multiple
          accept={ACCEPTED_FILE_TYPES}
          onChange={onFileSelect}
          className="hidden"
          disabled={isDisabled}
        />

        {variant === "compact" ? (
          <button
            type="button"
            onClick={onAddClick}
            className="text-muted hover:text-basis flex h-6 w-6 items-center justify-center rounded transition-colors hover:bg-canvasSubtle"
            disabled={isDisabled || atLimit}
            title={
              atLimit
                ? `Maximum ${MAX_ATTACHMENTS} attachments`
                : "Attach files"
            }
          >
            <RiAttachmentLine className="h-4 w-4" />
          </button>
        ) : (
          <Button
            type="button"
            onClick={onAddClick}
            appearance="outlined"
            kind="secondary"
            size="small"
            icon={<RiAttachmentLine className="h-4 w-4" />}
            iconSide="left"
            label="Add attachments"
            disabled={isDisabled || atLimit}
          />
        )}

        {showHelpText && (
          <p className="text-muted mt-1 text-xs">
            Max {MAX_ATTACHMENTS} files,{" "}
            {MAX_ATTACHMENT_SIZE_BYTES / (1024 * 1024)} MB each.
          </p>
        )}
      </div>
    </div>
  );
}
