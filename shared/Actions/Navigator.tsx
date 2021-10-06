import React, { useMemo } from "react";
import { Link } from "react-router-dom";
import styled from "@emotion/styled";
import { Action } from "src/types";
import Box from "src/shared/Box";
import { useIntegrations } from "src/scenes/Integrations/hooks";
import { useCurrentWorkspace } from "src/state/workspaces";
import { PageTitle } from "src/shared/Common";
import { useAccountIdentifier } from "src/scenes/Admin/Org/query";
import { categoryIcons, defaultIcon } from "./icons";

type Props = {
  actions: Action[];
  onClick?: (a: Action) => void;
};

const Navigator: React.FC<Props> = ({ actions, onClick }) => {
  const [identifier] = useAccountIdentifier();

  const w = useCurrentWorkspace();
  // Fetch our integrations so that we can get the integration name and data from
  // the DSN (eg. retriete stripe data from stripe.inngest.com)
  const integrations = useIntegrations();

  const actionIntegrations = useMemo(() => {
    return Object.values(integrations).filter((i) => {
      return !!i.methods.find((m) => m.for.includes("actions"));
    });
  }, [integrations]);

  const sorted = useMemo(() => {
    const copied = actions.slice();
    copied.sort((a, b) => a.latest.name.localeCompare(b.latest.name));
    return copied;
  }, [actions]);

  const builtin = useMemo(() => {
    return sorted.filter(
      (a) =>
        a.dsn.indexOf("com.inngest") === 0 || a.dsn.indexOf("inngest.com") === 0
    );
  }, [sorted]);

  const dsnPrefix = useMemo(() => {
    return identifier.data && identifier.data.account.identifier
      ? identifier.data.account.identifier.dsnPrefix
      : null;
  }, [identifier.data]);

  const custom: false | Array<Action> = useMemo(() => {
    if (!identifier.data) {
      return [];
    }
    if (!dsnPrefix) {
      return false;
    }
    return actions.filter((a) => a.dsn.indexOf(dsnPrefix) == 0);
  }, [actions, dsnPrefix]);

  return (
    <Wrapper>
      <Menu>
        <small>Custom</small>
        <a href="#your-own">Your own actions</a>
        <small>Inngest</small>
        <a href="#builtin">Built-in</a>
        <small>Integrations</small>
        {actionIntegrations.map((ai) => (
          <a href={`#${ai.service}`} key={ai.service}>
            {ai.name}
          </a>
        ))}
      </Menu>
      <div>
        <PageTitle style={{ padding: 0 }}>
          <div>
            <span>Actions</span>
            <h1>All actions</h1>
          </div>
        </PageTitle>

        <h2>
          <a id="builtin">Your own actions</a>
        </h2>
        {Array.isArray(custom) ? (
          custom.map((action: Action) => (
            <Items key={action.dsn}>
              <ActionFC action={action} onClick={onClick} />
            </Items>
          ))
        ) : (
          <>
            <p>
              You don't have any actions yet. You can run your own code, in any
              language, as a workflow or an internal task.
            </p>
            <a
              href="https://docs.inngest.com/docs/actions/serverless/tutorial"
              target="_blank"
              style={{ color: "inherit", textDecoration: "none" }}
            >
              <Box
                kind="dashed"
                style={{
                  marginBottom: "3rem",
                  background: "#fff",
                  padding: "36px 48px",
                  boxShadow: "0 5px 20px rgba(0, 0, 0, 0.05)",
                }}
              >
                <h3>Create your own actions</h3>
                <p>
                  Read our getting started guide which walks through how to make
                  your own actions →
                </p>
              </Box>
            </a>
          </>
        )}

        <h2>
          <a id="builtin">Built in</a>
        </h2>
        <Items>
          {builtin.map((action) => {
            if (
              !onClick &&
              (action.dsn === "inngest.com/email" ||
                action.dsn === "com.inngest/email")
            ) {
              return (
                <Link
                  to={`/workflows/actions/${encodeURIComponent(action.dsn)}`}
                  key={action.dsn}
                >
                  <ActionFC
                    action={action}
                    kind="hoverable"
                    onClick={onClick}
                  />
                </Link>
              );
            }
            return (
              <ActionFC action={action} key={action.dsn} onClick={onClick} />
            );
          })}
        </Items>

        {actionIntegrations.map((ai) => {
          const available = sorted.filter(
            (s) => s.dsn.indexOf(ai.service) === 0
          );

          if (available.length === 0) {
            return null;
          }

          return (
            <div>
              <h2>
                <a id={ai.service}>{ai.name}</a>
              </h2>
              <Items>
                {available.map((action) => (
                  <ActionFC
                    action={action}
                    key={action.dsn}
                    onClick={onClick}
                  />
                ))}
              </Items>
            </div>
          );
        })}
      </div>
    </Wrapper>
  );
};

export default Navigator;

const ActionFC: React.FC<{
  action: Action;
  kind?: any;
  onClick?: (a: Action) => void;
}> = ({ action, kind, onClick }) => {
  const Icon = categoryIcons[action.category.name] || defaultIcon;

  return (
    <ActionBox
      kind={onClick ? "hoverable" : kind || "plain"}
      onClick={() => onClick && onClick(action)}
    >
      <div>
        <Icon />
      </div>
      <div>
        <h3>{action.latest.name}</h3>
        <p>{action.tagline || "-"}</p>
      </div>
    </ActionBox>
  );
};

const Wrapper = styled.div`
  display: grid;
  grid-template-columns: 240px auto;
  grid-gap: 40px;
  padding-right: 40px;
  flex: 1;

  h2 a {
    color: inherit;
  }

  h2 + p {
    margin: -0.5rem 0 2rem;
    opacity: 0.6;
  }
`;

const Menu = styled.div`
  padding: 40px 40px;
  background: #f3f3f2;
  background: #fcfbf8;
  border-right: 1px solid #f6f6f4;

  small {
    display: block;
    text-transform: uppercase;
    margin: 2rem 0 0.5rem;
    opacity: 0.5;
  }

  h4 {
    margin: 0 0 1em;
  }

  input {
    display: inline;
    width: auto;
    margin-right: 0.75rem;
  }

  a,
  a + a {
    display: block;
    color: inherit;
    text-decoration: none;
    margin: 0.25rem 0;
    cursor: pointer;
  }

  hr {
    margin: 3rem 0;
  }
`;

const Items = styled.div`
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-gap: 40px;
  margin: 1rem 0 3rem;

  a {
    color: inherit;
    text-decoration: none;
  }
`;

const ActionBox = styled(Box)`
  display: grid;
  grid-template-columns: 1fr 6fr;
  grid-gap: 20px;
  line-height: 1.2;
  border: 1px solid #f4f4f4;
  align-items: center;
  height: 100%;
  padding: 16px 24px;

  > div:first-of-type {
    display: flex;
    align-self: stretch;
    align-items: center;
    justify-content: center;
    margin: 0 0 0.5rem 0;
  }

  svg {
    opacity: 0.7;
    stroke-width: 1;
  }

  h3 {
    margin: 3px 0 5px;
    padding: 0;
    font-weight: 600;
    line-height: 1.2;
    font-size: 14px;
  }

  p {
    opacity: 0.6;
    font-size: 13px;
    line-height: 1.2;
  }
`;
