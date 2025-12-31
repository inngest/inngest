import { useState, useMemo } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useServerFn } from "@tanstack/react-start";
import { useQuery } from "@tanstack/react-query";
import { useUser } from "@clerk/tanstack-react-start";
import { RiArrowLeftLine } from "@remixicon/react";
import { Button } from "@inngest/components/Button";
import { Textarea } from "@inngest/components/Forms/Textarea";
import { Alert } from "@inngest/components/Alert";
import { Select, type Option } from "@inngest/components/Select/Select";
import {
  formOptions,
  severityOptions,
  instructions,
  DEFAULT_BUG_SEVERITY_LEVEL,
  type TicketType,
  type BugSeverity,
} from "@/data/ticketOptions";
import { createTicket, getCustomerTierByEmail } from "@/data/plain";

export const Route = createFileRoute("/_authed/new")({
  component: NewTicketPage,
});

function NewTicketPage() {
  const navigate = useNavigate();
  const { user } = useUser();
  const createTicketFn = useServerFn(createTicket);
  const getCustomerTierFn = useServerFn(getCustomerTierByEmail);

  const [ticketType, setTicketType] = useState<TicketType>(null);
  const [body, setBody] = useState("");
  const [bugSeverity, setBugSeverity] = useState<BugSeverity>(
    DEFAULT_BUG_SEVERITY_LEVEL,
  );
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [result, setResult] = useState<{ ok?: boolean; message?: string }>({});

  // Fetch customer tier information
  const userEmail = user?.primaryEmailAddress?.emailAddress;
  const { data: tierInfo } = useQuery({
    queryKey: ["customerTier", userEmail],
    queryFn: () => getCustomerTierFn({ data: { email: userEmail! } }),
    enabled: !!userEmail,
  });

  const isEnterprise = tierInfo?.isEnterprise ?? false;
  const isPaid = tierInfo?.isPaid ?? false;

  // Convert form options to Select options format (memoized for stable refs)
  const ticketTypeOptions: Option[] = useMemo(
    () =>
      formOptions.map((opt) => ({
        id: opt.value,
        name: opt.label,
      })),
    [],
  );

  const selectedTypeOption = useMemo(
    () =>
      ticketType
        ? ticketTypeOptions.find((opt) => opt.id === ticketType) || null
        : null,
    [ticketType, ticketTypeOptions],
  );

  // Convert severity options to Select options format based on tier
  const severitySelectOptions: Option[] = useMemo(
    () =>
      severityOptions.map((opt) => ({
        id: opt.value,
        name: `${opt.label} - ${opt.description}`,
        // Disable if: enterpriseOnly and not enterprise, or paidOnly and not paid
        disabled:
          (opt.enterpriseOnly && !isEnterprise) || (opt.paidOnly && !isPaid),
      })),
    [isEnterprise, isPaid],
  );

  const selectedSeverityOption = useMemo(
    () => severitySelectOptions.find((opt) => opt.id === bugSeverity) || null,
    [bugSeverity, severitySelectOptions],
  );

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!ticketType || !body.trim()) {
      setResult({ ok: false, message: "Please fill in all required fields" });
      return;
    }

    if (!user?.primaryEmailAddress?.emailAddress) {
      setResult({ ok: false, message: "Unable to get user email" });
      return;
    }

    setIsSubmitting(true);
    setResult({});

    try {
      const response = await createTicketFn({
        data: {
          user: {
            id: user.externalId || user.id,
            email: user.primaryEmailAddress.emailAddress,
            name: user.fullName || undefined,
          },
          ticket: {
            type: ticketType,
            body: body.trim(),
            severity: ticketType === "bug" ? bugSeverity : undefined,
          },
        },
      });

      if (response.success) {
        setResult({
          ok: true,
          message: "Support ticket created successfully!",
        });
        // Reset form
        setTicketType(null);
        setBody("");
        setBugSeverity(DEFAULT_BUG_SEVERITY_LEVEL);
        // Navigate to home after a short delay
        setTimeout(() => {
          navigate({ to: "/" });
        }, 1500);
      } else {
        setResult({
          ok: false,
          message:
            response.error ||
            "Failed to create support ticket. Please try again.",
        });
      }
    } catch (error) {
      console.error("Error creating ticket:", error);
      setResult({
        ok: false,
        message:
          "Failed to create support ticket. Please email hello@inngest.com if the problem persists.",
      });
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen">
      {/* Back button */}
      <Link
        to="/"
        className="text-muted hover:text-basis mb-6 inline-flex items-center gap-2 text-sm font-medium transition-colors"
      >
        <RiArrowLeftLine className="h-4 w-4" />
        Back to tickets
      </Link>

      <div className="mx-auto max-w-2xl mb-8">
        <div className="mb-8">
          <h1 className="text-basis mb-2 text-2xl font-bold">
            Create Support Ticket
          </h1>
          <p className="text-muted text-sm">
            Tell us how we can help. Our team will respond as soon as possible.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-6">
          {/* Ticket Type */}
          <div className="flex flex-col gap-2">
            <label className="text-basis text-sm font-medium">
              What do you need help with?
            </label>
            <Select
              label="Type"
              isLabelVisible={false}
              value={selectedTypeOption}
              onChange={(option: Option) =>
                setTicketType(option.id as TicketType)
              }
              className="w-full"
            >
              <Select.Button>
                {selectedTypeOption?.name || "Select a topic..."}
              </Select.Button>
              <Select.Options className="w-full">
                {ticketTypeOptions.map((option) => (
                  <Select.Option key={option.id} option={option}>
                    {option.name}
                  </Select.Option>
                ))}
              </Select.Options>
            </Select>
          </div>

          {/* Message */}
          <div className="flex flex-col gap-2">
            <label className="text-basis text-sm font-medium">
              {ticketType ? instructions[ticketType] : "Describe your issue"}
            </label>
            <Textarea
              placeholder="Describe your issue..."
              value={body}
              onChange={setBody}
              rows={6}
              required
            />
          </div>

          {/* Severity (only for bugs) */}
          {ticketType === "bug" && (
            <div className="flex flex-col gap-2">
              <label className="text-basis text-sm font-medium">
                How severe is your issue?
              </label>
              <Select
                label="Severity"
                isLabelVisible={false}
                value={selectedSeverityOption}
                onChange={(option: Option) =>
                  setBugSeverity(option.id as BugSeverity)
                }
                className="w-full"
              >
                <Select.Button>
                  {selectedSeverityOption?.name || "Select severity..."}
                </Select.Button>
                <Select.Options className="w-full">
                  {severitySelectOptions.map((option) => (
                    <Select.Option key={option.id} option={option}>
                      {option.name}
                    </Select.Option>
                  ))}
                </Select.Options>
              </Select>
              <p className="text-muted text-xs">
                Some severity levels are only available on paid plans.
              </p>
            </div>
          )}

          {/* Submit Button */}
          <Button
            type="submit"
            kind="primary"
            label={isSubmitting ? "Creating..." : "Create Support Ticket"}
            disabled={isSubmitting || !ticketType || !body.trim()}
            className="w-full"
          />

          {/* Result Message */}
          {result.message && (
            <Alert severity={result.ok ? "info" : "error"}>
              {result.message}
            </Alert>
          )}

          {/* Help Text */}
          <p className="text-muted text-center text-sm">
            Our team will respond via email as soon as possible based on the
            severity of your issue.
          </p>
        </form>
      </div>
    </div>
  );
}
