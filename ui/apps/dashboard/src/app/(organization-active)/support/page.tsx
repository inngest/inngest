"use client";

import { useEffect, useState } from "react";
import { type Route } from "next";
import { useSearchParams } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { Banner } from "@inngest/components/Banner";
import { Button } from "@inngest/components/Button";
import { Link } from "@inngest/components/Link";
import { Pill } from "@inngest/components/Pill/Pill";
import { cn } from "@inngest/components/utils/classNames";
import { RiArrowLeftLine, RiGithubFill } from "@remixicon/react";
import {
  ThreadStatus,
  type ThreadPartsFragment,
} from "@team-plain/typescript-sdk";
import { useQuery } from "urql";

import { isEnterprisePlan } from "@/components/Billing/Plans/utils";
import { graphql } from "@/gql";
import LoadingIcon from "@/icons/LoadingIcon";
import { SupportForm } from "./SupportForm";
import { useSystemStatus } from "./statusPage";
import { type TicketType } from "./ticketOptions";

const GetAccountSupportInfoDocument = graphql(`
  query GetAccountSupportInfo {
    account {
      id
      plan {
        id
        name
        amount
        features
      }
    }
  }
`);

export default function Page() {
  const status = useSystemStatus();
  const { isSignedIn } = useAuth();
  const searchParams = useSearchParams();
  const [{ data, fetching }] = useQuery({
    query: GetAccountSupportInfoDocument,
    pause: !isSignedIn,
  });

  const plan = data?.account.plan;
  const isEnterprise = plan ? isEnterprisePlan(plan) : false;
  const isPaid = (plan?.amount || 0) > 0 || isEnterprise;
  const preselectedTicketType = searchParams.get("q") as TicketType;

  return (
    <div className="h-full overflow-y-scroll">
      <div className="mx-auto max-w-screen-xl px-6">
        <div className="my-4 inline-block">
          <Button
            href={process.env.NEXT_PUBLIC_HOME_PATH as Route}
            size="small"
            appearance="outlined"
            kind="secondary"
            icon={<RiArrowLeftLine />}
            iconSide="left"
            label={isSignedIn ? "Back To Dashboard" : "Sign In To Dashboard"}
          />
        </div>
        {/* Thanksgiving banner for limited support availability, won't show after November 29 and will removed this after */}
        {new Date() < new Date("2025-11-29") && (
          <Banner severity="info" showIcon={false}>
            In observance of Thanksgiving, our support team will have limited
            availability from November 26–28. Thank you for your patience as
            response times may be delayed.
          </Banner>
        )}
        <header className="border-subtle flex items-center justify-between border-b py-6">
          <h1 className="text-2xl font-semibold">Inngest Support</h1>
          <div className="" title={`Status updated at ${status.updated_at}`}>
            <a
              href={status.url}
              target="_blank"
              className="hover:text-link bg-canvasSubtle hover:bg-canvasMuted flex items-center gap-2 rounded px-3 py-1.5 text-sm font-medium"
            >
              <span
                className={`mx-1 inline-flex h-2.5 w-2.5 rounded-full`}
                style={{ backgroundColor: status.indicatorColor }}
              ></span>
              {status.description}
            </a>
          </div>
        </header>
        <div className="my-8 grid gap-12 md:grid-cols-2">
          <SupportChannel
            title="Create a ticket"
            label="Account required"
            className="min-h-[384px]"
          >
            {fetching ? (
              <div className="mt-4 flex w-full place-content-center">
                <LoadingIcon />
              </div>
            ) : !isSignedIn ? (
              <>
                <p>Sign in or sign up for an account to create a ticket.</p>
                <div className="flex gap-2">
                  <Button
                    kind="primary"
                    href={
                      `${process.env.NEXT_PUBLIC_SIGN_IN_PATH}?ref=support` as Route
                    }
                    label="Sign In"
                  />
                  <Button
                    kind="primary"
                    href={
                      `${process.env.NEXT_PUBLIC_SIGN_UP_PATH}?ref=support` as Route
                    }
                    label="Sign Up"
                  />
                </div>
              </>
            ) : (
              <>
                <SupportForm
                  isEnterprise={isEnterprise}
                  isPaid={isPaid}
                  preselectedTicketType={preselectedTicketType}
                />
                <SupportTickets isSignedIn={isSignedIn} />
              </>
            )}
          </SupportChannel>
          <SupportChannel title="Live chat" label="Enterprise">
            {fetching ? (
              <div className="mt-4 flex w-full place-content-center">
                <LoadingIcon />
              </div>
            ) : isEnterprise ? (
              <p>
                Create a general request support ticket to request a dedicated
                Slack or Discord channel with the Inngest team.
              </p>
            ) : (
              <p>
                Enterprise plans include live chat support including dedicated
                Slack channel and support SLAs. To chat with someone about our
                enterprise plans,{" "}
                <Link
                  size="medium"
                  target="_blank"
                  href="https://inngest.com/contact"
                  className="inline"
                >
                  fill out the form here
                </Link>
                .
              </p>
            )}
          </SupportChannel>

          <SupportChannel title="Community">
            <p>
              Chat with other developers and the Inngest team in our{" "}
              <Link
                target="_blank"
                href="https://www.inngest.com/discord"
                className="inline-flex"
                size="medium"
              >
                Discord community
              </Link>
              . Search for topics and questions in our{" "}
              <Link
                href="https://discord.com/channels/842170679536517141/1051516534029291581"
                className="inline-flex"
                target="_blank"
                size="medium"
              >
                #help-forum
              </Link>{" "}
              channel or submit your own question.
            </p>
            <Button
              kind="primary"
              href="https://www.inngest.com/discord"
              target="_blank"
              label="Join our Discord"
            />
          </SupportChannel>
          <SupportChannel title="Open Source">
            <p>File an issue in our open source repos on Github:</p>
            <div>
              <p className="mb-2 text-sm font-medium">
                Inngest CLI + Dev Server
              </p>
              <Button
                appearance="outlined"
                kind="secondary"
                href="https://github.com/inngest/inngest/issues"
                label="inngest/inngest"
                icon={<RiGithubFill />}
                className="justify-start"
                iconSide="left"
              />
            </div>
            <div>
              <p className="mb-2 text-sm font-medium">SDKs</p>
              <Button
                appearance="outlined"
                kind="secondary"
                href="https://github.com/inngest/inngest-js/issues"
                label="inngest/inngest-js"
                icon={<RiGithubFill />}
                iconSide="left"
              />
            </div>
          </SupportChannel>
        </div>
        <Footer />
      </div>
    </div>
  );
}

function SupportChannel({
  title,
  label,
  className = "",
  children,
}: {
  title: string;
  label?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("flex flex-col items-start gap-6 leading-7", className)}>
      <h2 className="flex items-center gap-4 text-lg font-semibold">
        {title}
        {label && <Pill>{label}</Pill>}
      </h2>
      {children}
    </div>
  );
}

function SupportTickets({ isSignedIn }: { isSignedIn?: boolean }) {
  const [isFetchingTickets, setIsFetchingTickets] = useState(false);
  const [tickets, setTickets] = useState<ThreadPartsFragment[]>([]);
  useEffect(
    function () {
      async function fetchTickets() {
        setIsFetchingTickets(true);
        const result = await fetch(`/api/support-tickets`, {
          method: "GET",
          credentials: "include",
          redirect: "error",
        });
        const body = await result.json();
        if (body) {
          setIsFetchingTickets(false);
          setTickets(body.data);
        }
      }
      fetchTickets();
    },
    [isSignedIn],
  );

  return isFetchingTickets ? (
    <LoadingIcon />
  ) : (
    <div className="w-full">
      <h3 className="mb-2 text-base font-semibold">Recent tickets</h3>
      <div className="border-muted divide-subtle grid w-full grid-cols-1 divide-y rounded-md border text-sm">
        {tickets.length > 0 ? (
          tickets.map((ticket) => (
            <div key={ticket.id} className="flex items-center gap-2 px-2 py-2">
              {ticket.status === ThreadStatus.Done ? (
                <Pill kind="primary">Closed</Pill>
              ) : (
                <Pill kind="info">Open</Pill>
              )}
              <span
                className="grow overflow-hidden text-ellipsis whitespace-nowrap"
                title={ticket.previewText || ticket.title}
              >
                {ticket.previewText || ticket.title}
              </span>
              <span className="flex gap-2">
                {ticket.labels.map((label) => (
                  <Pill
                    appearance="outlined"
                    key={label.id}
                    className="whitespace-nowrap"
                  >
                    {label.labelType.name}
                  </Pill>
                ))}
              </span>
            </div>
          ))
        ) : (
          <div className="px-2 py-2">No open tickets</div>
        )}
      </div>
    </div>
  );
}

const FOOTER_NAV_ITEMS = [
  {
    name: "Documentation",
    url: "https://www.inngest.com/docs?ref=support-center",
  },
  {
    name: "Privacy",
    url: "https://www.inngest.com/privacy?ref=support-center",
  },
  {
    name: "Terms & Conditions",
    url: "https://www.inngest.com/terms?ref=support-center",
  },
  {
    name: "Security",
    url: "https://www.inngest.com/security?ref=support-center",
  },
];

function Footer() {
  return (
    <div className="border-subtle text-subtle mt-32 flex flex-col items-center justify-between gap-8 border-t py-6 text-sm md:flex-row">
      <div>© {new Date().getFullYear()} Inngest, Inc. All rights reserved.</div>
      <div className="flex flex-row gap-4">
        {FOOTER_NAV_ITEMS.map((i) => (
          <a href={i.url} key={i.name} className="hover:text-link">
            {i.name}
          </a>
        ))}
      </div>
    </div>
  );
}
