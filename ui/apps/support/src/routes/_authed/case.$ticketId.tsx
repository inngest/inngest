import { useEffect, useMemo, useRef, useState } from "react";
import { Link, createFileRoute, useRouter } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useServerFn } from "@tanstack/react-start";
import { useUser } from "@clerk/tanstack-react-start";
import {
  RiArrowLeftLine,
  RiArrowRightUpLine,
  RiSlackLine,
  RiUserLine,
} from "@remixicon/react";
import { Button } from "@inngest/components/Button";
import { formatDistanceToNow } from "date-fns";
import { InngestLogoSmall } from "@inngest/components/icons/logos/InngestLogoSmall";
import { Image } from "@unpic/react";
import type { TicketDetail, TimeLineEntryEdge } from "@/data/plain";
import {
  getTicketById,
  getTimelineEntriesForTicket,
  replyToThread,
  closeTicket,
} from "@/data/plain";
import { Main } from "@/components/Main";
import { Markdown } from "@/components/Markdown/Markdown";
import { PriorityBadge, StatusBadge } from "@/components/Support/TicketBadges";
import { ChannelBadge } from "@/components/Support/ChannelBadge";
import { formatTimestamp } from "@/utils/ticket";
import { Attachment } from "@/components/Support/Attachment";
import {
  AttachmentUploadField,
  useAttachmentUpload,
} from "@/components/Support/AttachmentUploadField";

export const Route = createFileRoute("/_authed/case/$ticketId")({
  component: TicketDetailPage,
  loader: async ({ params }) => {
    const [ticket, timelineEntries] = await Promise.all([
      getTicketById({ data: { ticketId: params.ticketId } }),
      getTimelineEntriesForTicket({
        data: { ticketId: params.ticketId },
      }),
    ]);

    return { ticket, timelineEntries };
  },
});

function TicketDetailPage() {
  const { ticket, timelineEntries: initialTimelineEntries } =
    Route.useLoaderData();
  const params = Route.useParams();
  const { user } = useUser();
  const timelineEndRef = useRef<HTMLDivElement>(null);

  const { data: serverEntries, refetch: refetchTimeline } = useQuery({
    queryKey: ["timeline", params.ticketId],
    queryFn: () =>
      getTimelineEntriesForTicket({ data: { ticketId: params.ticketId } }),
    initialData: initialTimelineEntries,
    staleTime: 30_000,
  });

  const [pendingEntries, setPendingEntries] = useState<TimeLineEntryEdge[]>([]);

  const timelineEntries = useMemo(() => {
    const real = serverEntries ?? [];
    const pending = pendingEntries.filter((pending) => {
      const pendingTime = new Date(pending.node.timestamp.iso8601).getTime();
      return !real.some(
        (real) =>
          real.node.actor.__typename === "CustomerActor" &&
          new Date(real.node.timestamp.iso8601).getTime() >= pendingTime,
      );
    });
    return [...real, ...pending];
  }, [serverEntries, pendingEntries]);

  if (!ticket || !timelineEntries) {
    return <div>Error loading ticket</div>;
  }

  // Check if this is a Slack conversation
  const isSlackChannel = ticket.channel === "SLACK";
  const userEmail = user?.primaryEmailAddress?.emailAddress;

  const scrollToBottom = () => {
    setTimeout(() => {
      timelineEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, 100);
  };

  // Find the first Slack message link for the "Reply in Slack" button
  const slackLink = timelineEntries.find(
    (entry) =>
      entry.node.entry.__typename === "SlackMessageEntry" ||
      entry.node.entry.__typename === "SlackReplyEntry",
  )?.node.entry;
  const slackMessageLink =
    slackLink &&
    (slackLink.__typename === "SlackMessageEntry" ||
      slackLink.__typename === "SlackReplyEntry")
      ? slackLink.slackMessageLink
      : undefined;

  return (
    <Main className="min-h-screen lg:max-w-6xl">
      <div className="mb-4 lg:`mb-8 max-w-4xl">
        {/* Back button */}
        <Link
          to="/"
          className="text-muted hover:text-basis mb-6 inline-flex items-center gap-2 text-sm font-medium transition-colors"
        >
          <RiArrowLeftLine className="h-4 w-4" />
          Back to tickets
        </Link>

        {/* Title and Status */}
        <header className="flex items-center justify-between">
          <h1 className="text-basis font-medium text-lg leading-4">
            {ticket.title}
          </h1>
          <StatusBadge status={ticket.status} size="md" />
        </header>
      </div>

      <div className="flex flex-col flex-grow h-full lg:flex-row gap-10">
        <div className="flex flex-col flex-grow max-w-4xl">
          {/* Ticket header */}
          <div className="mb-8 flex flex-col gap-2 pb-2 pt-2 text-sm md:text-base lg:hidden">
            <Metadata ticket={ticket} />
          </div>

          {/* Conversation timeline */}
          <div className="flex-1 space-y-8">
            {timelineEntries.length === 0 ? (
              <div className="bg-canvasSubtle border-subtle rounded-xl border p-12 text-center">
                <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-canvasMuted">
                  <svg
                    className="text-muted h-8 w-8"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
                    />
                  </svg>
                </div>
                <p className="text-basis mb-1 text-lg font-medium">
                  No messages yet
                </p>
                <p className="text-muted text-sm">
                  The conversation will appear here once messages are exchanged.
                </p>
              </div>
            ) : (
              <div className="flex flex-col gap-0">
                {" "}
                {/* To support threading, we handle spacing in the element */}
                {timelineEntries.map((entry, idx, arr) => {
                  // If multiple messages are send from Slack within 2 minutes of each, thread them together
                  const entryTypename = entry.node.entry.__typename;
                  const isSlackMessage =
                    entryTypename === "SlackMessageEntry" ||
                    entryTypename === "SlackReplyEntry";
                  const previousEntry = arr[idx - 1] as
                    | typeof entry
                    | undefined;
                  const prevTypename = previousEntry?.node.entry.__typename;
                  const isPreviousSlackMessage = prevTypename
                    ? prevTypename === "SlackMessageEntry" ||
                      prevTypename === "SlackReplyEntry"
                    : false;
                  const shouldThread =
                    isSlackMessage &&
                    isPreviousSlackMessage &&
                    previousEntry !== undefined &&
                    new Date(entry.node.timestamp.iso8601).getTime() -
                      new Date(previousEntry.node.timestamp.iso8601).getTime() <
                      2 * 60 * 1000;
                  return (
                    <TimelineEntry
                      key={entry.node.id}
                      entry={entry}
                      idx={idx}
                      shouldThread={shouldThread}
                    />
                  );
                })}
                {/* Scroll target for after sending a message */}
                <div ref={timelineEndRef} />
              </div>
            )}
          </div>

          {/* Reply form or Slack button */}
          {isSlackChannel && slackMessageLink ? (
            <div className="sticky bottom-0 border-t border-muted bg-canvasBase py-2">
              <Button
                kind="primary"
                appearance="outlined"
                href={slackMessageLink}
                target="_blank"
                label="Reply in Slack"
                icon={<RiSlackLine className="h-4 w-4" />}
                iconSide="left"
              />
            </div>
          ) : userEmail ? (
            <ReplyForm
              ticketId={ticket.id}
              userEmail={userEmail}
              onSuccess={(sentMessage: string) => {
                const now = new Date().toISOString();
                const optimistic: TimeLineEntryEdge = {
                  cursor: `pending-${Date.now()}`,
                  node: {
                    id: `pending-${Date.now()}`,
                    timestamp: { __typename: "DateTime", iso8601: now },
                    actor: {
                      __typename: "CustomerActor",
                      customer: {
                        fullName: user?.fullName || userEmail || "You",
                        avatarUrl: user?.imageUrl || "",
                        email: { email: userEmail || "" },
                      },
                    },
                    entry: {
                      __typename: "EmailEntry",
                      emailId: `pending-${Date.now()}`,
                      subject: "",
                      textContent: sentMessage,
                      markdownContent: sentMessage,
                      hasMoreMarkdownContent: false,
                      fullMarkdownContent: sentMessage,
                      sentAt: {
                        unixTimestamp: String(Math.floor(Date.now() / 1000)),
                        iso8601: now,
                      },
                      attachments: [],
                    },
                  },
                };
                setPendingEntries((prev) => [...prev, optimistic]);
                scrollToBottom();
                setTimeout(() => refetchTimeline(), 3000);
                setTimeout(() => refetchTimeline(), 8000);
              }}
            />
          ) : null}
        </div>
        <div className="hidden lg:block">
          <Metadata ticket={ticket} />
        </div>
      </div>
    </Main>
  );
}

function Metadata({ ticket }: { ticket: TicketDetail }) {
  const router = useRouter();
  const [isClosing, setIsClosing] = useState(false);
  const [closeError, setCloseError] = useState<string | null>(null);
  const closeTicketFn = useServerFn(closeTicket);

  async function handleCloseTicket() {
    if (!ticket) return;
    setIsClosing(true);
    setCloseError(null);
    try {
      const result = await closeTicketFn({
        data: { threadId: ticket.id },
      });
      if (result.success) {
        await router.invalidate();
      } else {
        setCloseError(result.error || "Failed to close ticket");
      }
    } catch (err) {
      console.error("Error closing ticket:", err);
      setCloseError("Failed to close ticket. Please try again.");
    } finally {
      setIsClosing(false);
    }
  }

  const isOpen = ticket.status.toLowerCase() !== "done";
  return (
    <aside className="flex flex-col gap-2 lg:gap-4">
      {/* Ticket number */}
      <div className="flex md:flex-col items-center md:items-start gap-2 leading-4">
        <span className="text-muted text-sm">Ticket number:</span>
        <span className="text-basis font-mono">{ticket.ref}</span>
      </div>

      {/* Priority */}
      <div className="flex md:flex-col items-center md:items-start gap-2 leading-4">
        <span className="text-muted text-sm">Priority:</span>
        <PriorityBadge priority={ticket.priority} />
      </div>

      {/* Source */}
      {ticket.channel && (
        <div className="flex md:flex-col items-center md:items-start gap-2 leading-4">
          <span className="text-muted text-sm">Source:</span>
          <ChannelBadge channel={ticket.channel} showLabel={true} />
        </div>
      )}

      {/* Created */}
      <div className="flex md:flex-col items-center md:items-start gap-2 leading-4">
        <span className="text-muted text-sm">Created:</span>
        <span className="text-basis leading-4">
          {formatTimestamp(ticket.createdAt)}
        </span>
      </div>

      {/* Updated */}
      <div className="flex md:flex-col items-center md:items-start gap-2 leading-4">
        <span className="text-muted text-sm">Updated:</span>
        <span className="text-basis leading-4">
          {formatTimestamp(ticket.updatedAt)}
        </span>
      </div>

      {/* Close Ticket */}
      {isOpen && (
        <div className="flex items-center gap-2 pt-2">
          <Button
            kind="danger"
            appearance="outlined"
            size="small"
            label={isClosing ? "Closing..." : "Close ticket"}
            disabled={isClosing}
            onClick={handleCloseTicket}
          />
          {closeError && (
            <span className="text-sm text-red-500">{closeError}</span>
          )}
        </div>
      )}
    </aside>
  );
}

function ReplyForm({
  ticketId,
  userEmail,
  onSuccess,
}: {
  ticketId: string;
  userEmail: string;
  onSuccess: (message: string) => void;
}) {
  const [message, setMessage] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const replyToThreadFn = useServerFn(replyToThread);
  const {
    attachments,
    isUploading,
    uploadedAttachmentIds,
    fileInputRef,
    handleFileSelect,
    removeAttachment,
    openFilePicker,
    clearAttachments,
  } = useAttachmentUpload({
    userEmail,
    onError: setError,
  });
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
  }, [message]);

  const hasContent =
    message.trim().length > 0 || uploadedAttachmentIds.length > 0;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!hasContent) return;

    setIsSubmitting(true);
    setError(null);

    try {
      const result = await replyToThreadFn({
        data: {
          threadId: ticketId,
          message: message.trim(),
          userEmail,
          attachmentIds:
            uploadedAttachmentIds.length > 0
              ? uploadedAttachmentIds
              : undefined,
        },
      });

      if (result.success) {
        const sentMessage = message.trim();
        setMessage("");
        clearAttachments();
        onSuccess(sentMessage);
      } else {
        setError(result.error || "Failed to send message");
      }
    } catch (err) {
      console.error("Error sending reply:", err);
      setError("Failed to send message. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="sticky bottom-0 bg-gradient-to-t from-white/50 via-white/50 to-transparent pb-4 pt-6">
      <form onSubmit={handleSubmit}>
        <div className="border-muted bg-canvasBase flex flex-col gap-2 rounded-lg border px-4 py-3 shadow-sm">
          <textarea
            ref={textareaRef}
            placeholder="Add new message"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            rows={1}
            className="text-basis placeholder:text-disabled min-h-[21px] w-full resize-none overflow-hidden border-0 bg-transparent p-0 text-sm leading-5 outline-none focus:ring-0"
            disabled={isSubmitting}
          />

          <div className="flex items-center justify-between">
            <div>
              <AttachmentUploadField
                attachments={attachments}
                isUploading={isUploading}
                isSubmitting={isSubmitting}
                fileInputRef={fileInputRef}
                onFileSelect={handleFileSelect}
                onRemoveAttachment={removeAttachment}
                onAddClick={openFilePicker}
                variant="compact"
                showHelpText={false}
              />
            </div>
            <div>
              <Button
                type="submit"
                kind="primary"
                appearance="solid"
                size="small"
                label={isSubmitting ? "Sending..." : "Submit"}
                icon={<RiArrowRightUpLine className="h-4 w-4" />}
                disabled={isSubmitting || isUploading || !hasContent}
                className="h-6 px-2 text-xs"
              />
            </div>
          </div>
        </div>
        {error && <p className="mt-2 text-sm text-red-500">{error}</p>}
        <p className="text-muted mt-1 text-xs">
          Max 5 files, 10 MB each. PDF, images, and common document types
          accepted.
        </p>
      </form>
    </div>
  );
}

function TimelineEntry({
  entry,
  idx,
  shouldThread,
}: {
  entry: TimeLineEntryEdge;
  shouldThread: boolean;
  idx: number;
}) {
  const actorTypename = entry.node.actor.__typename;
  const isStaff =
    actorTypename === "UserActor" || actorTypename === "MachineUserActor";
  const staffName =
    actorTypename === "UserActor"
      ? entry.node.actor.user.fullName
      : actorTypename === "MachineUserActor"
      ? entry.node.actor.machineUser.fullName
      : "";
  const actorName =
    actorTypename === "CustomerActor"
      ? entry.node.actor.customer.fullName || "Customer"
      : isStaff
      ? staffName || "Inngest Support Team"
      : "Unknown";

  const timeAgo = formatDistanceToNow(new Date(entry.node.timestamp.iso8601), {
    addSuffix: true,
  });

  const messageContent =
    entry.node.entry.__typename === "EmailEntry"
      ? entry.node.entry.markdownContent
      : entry.node.entry.__typename === "CustomEntry"
      ? entry.node.entry.components
          .map((component) => component.text)
          .join("\n")
      : entry.node.entry.text || "";

  return (
    <div
      key={entry.node.id}
      className={`flex flex-col gap-3 ${
        !shouldThread && idx !== 0 ? "mt-8" : "mt-0"
      }`}
    >
      {!shouldThread && (
        <div className="flex items-center gap-2">
          {/* Avatar */}
          <div
            className={`flex h-6 w-6 items-center justify-center overflow-hidden rounded-full ${
              isStaff
                ? "bg-basis text-alwaysWhite"
                : "bg-secondary-moderate text-alwaysWhite"
            }`}
          >
            {isStaff ? (
              <div className="flex h-6 w-6 p-1 items-center justify-center bg-contrast">
                <InngestLogoSmall className="h-5 w-5 text-alwaysWhite" />
              </div>
            ) : entry.node.actor.__typename === "CustomerActor" &&
              entry.node.actor.customer.avatarUrl ? (
              <Image
                src={entry.node.actor.customer.avatarUrl}
                className="h-6 w-6 rounded-full object-cover"
                width={24}
                height={24}
                alt="User avatar"
              />
            ) : (
              <RiUserLine className="h-4 w-4" />
            )}
          </div>
          <span className="text-basis text-sm font-medium leading-5">
            {actorName}
          </span>
          {actorTypename === "CustomerActor" &&
            // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
            entry.node.actor.customer.email &&
            !entry.node.actor.customer.email.email.match(
              /@plain-customer\.com$/,
            ) && (
              <span className="text-muted text-sm leading-5">
                {entry.node.actor.customer.email.email}
              </span>
            )}
          <span
            className="text-muted text-sm leading-5"
            title={entry.node.timestamp.iso8601}
          >
            {timeAgo}
          </span>
          {(entry.node.entry.__typename === "SlackMessageEntry" ||
            entry.node.entry.__typename === "SlackReplyEntry") && (
            <a
              href={entry.node.entry.slackMessageLink}
              className="text-muted text-sm leading-5 hover:text-basis"
              title="View on Slack"
            >
              <RiSlackLine className="h-4 w-4" />
            </a>
          )}
        </div>
      )}

      {/* Message content */}
      <div className="text-basis text-sm">
        <Markdown content={messageContent} />

        {(entry.node.entry.__typename === "EmailEntry" ||
          entry.node.entry.__typename === "SlackMessageEntry" ||
          entry.node.entry.__typename === "SlackReplyEntry") &&
          entry.node.entry.attachments.length > 0 && (
            <div className="flex flex-row flex-wrap gap-1">
              {entry.node.entry.attachments.map((attachment) => (
                <Attachment key={attachment.id} attachmentId={attachment.id} />
              ))}
            </div>
          )}
      </div>
    </div>
  );
}
