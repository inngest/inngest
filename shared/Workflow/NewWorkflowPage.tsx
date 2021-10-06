import React, { useState, useMemo, useEffect } from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";
import cronstrue from "cronstrue";

import Button, { ButtonGroup } from "src/shared/Button";
import ShapeDisplay from "src/shared/ShapeDisplay";
import EventIcon from "src/shared/Icons/Event";
import RadioBox from "src/shared/RadioBox";
import Search, { ResultProps } from "src/shared/Search/Search";
import { useEventNames, useEventDetails, Event } from "./queries";

type Props = {
  visible?: boolean;
  onSubmit: (
    name: string,
    triggerType: "cron" | "event",
    triggerValue: string
  ) => void;
};

const NewWorkflowPage = (props: Props) => {
  // Autofill the trigger if we already have a selector in query parameters.
  const params = new URLSearchParams(window.location.search);
  const initialType = ["cron", "event"].includes(params.get("type") || "")
    ? params.get("type")
    : "event";
  const [triggerType, setTriggerType] = useState<"cron" | "event">(
    initialType as "cron" | "event"
  );
  const [triggerValue, setTriggerValue] = useState<string>(
    params.get("value") || ""
  );
  const [workflowName, setWorkflowName] = useState<string>("");

  const { visible, onSubmit } = props;

  const onContinue = (e: React.SyntheticEvent) => {
    e.preventDefault();
    if (!triggerValue) return;
    onSubmit(workflowName, triggerType, triggerValue);
  };

  const onTriggerTypeChange = (newType: "cron" | "event") => {
    if (newType !== triggerType) {
      setTriggerType(newType);
      setTriggerValue("");
    }
  };

  if (!visible) {
    return null;
  }

  return (
    <Wrapper>
      <Header>
        <ButtonGroup right style={{ flex: 1, alignSelf: "center" }}>
          <Button kind="submit" disabled={!triggerValue} onClick={onContinue}>
            Continue
          </Button>
        </ButtonGroup>
      </Header>
      <Content>
        <h2 style={{ marginTop: "15px" }}>New workflow</h2>
        <label>
          <h3>Name</h3>
          <input
            type="text"
            placeholder="My workflow"
            value={workflowName}
            onChange={(e) => setWorkflowName(e.target.value)}
          />
        </label>
        <TriggerSelection value={triggerType} onChange={onTriggerTypeChange} />
        {triggerType === "event" && (
          <EventTrigger onChange={setTriggerValue} value={triggerValue} />
        )}
        {triggerType === "cron" && <CRONTrigger onChange={setTriggerValue} />}

        <ButtonGroup style={{ flex: 0, marginTop: "20px" }}>
          <Button kind="submit" disabled={!triggerValue} onClick={onContinue}>
            Continue
          </Button>
        </ButtonGroup>
      </Content>
    </Wrapper>
  );
};

type TriggerSelectionProps = {
  value: "cron" | "event";
  onChange: (newValue: "cron" | "event") => void;
};

const TriggerDescription = styled.div`
  text-align: left;
  width: 250px;

  span {
    display: block;
    opacity: 0.6;
  }
`;

const TriggerSelection: React.FC<TriggerSelectionProps> = ({
  value,
  onChange,
}) => {
  return (
    <div>
      <h3>Select a trigger</h3>
      <div style={{ display: "flex" }}>
        <RadioBox
          style={{ marginRight: "20px" }}
          value="event"
          checked={value === "event"}
          onChange={(value) => onChange(value as "event" | "cron")}
        >
          <TriggerDescription>
            <b>Event</b>
            <span>Run this workflow when Inngest receives an event</span>
          </TriggerDescription>
        </RadioBox>
        <RadioBox
          value="cron"
          checked={value === "cron"}
          onChange={(value) => onChange(value as "event" | "cron")}
        >
          <TriggerDescription>
            <b>Scheduled</b>
            <span>
              Run this <b>cron</b> workflow on a timed basis
            </span>
          </TriggerDescription>
        </RadioBox>
      </div>
    </div>
  );
};

const CRONExamples = styled.div`
  margin-top: 20px;
  align-self: flex-start;

  > small {
    display: grid;
    grid-template-columns: 200px auto;
    gap: 10px 0;
  }

  code {
    background: #d7dfd599;
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 12px;
  }

  p {
    margin-bottom: 20px;
  }

  p,
  small {
    opacity: 0.9;
  }
`;

const CronContainer = styled.div`
  margin-top: 20px;
  align-self: flex-start;

  display: flex;
  flex-flow: column;

  input {
    width: 680px;
    margin-bottom: 5px;
  }
`;

type CRONTriggerProps = { onChange: (cron: string) => void; value?: string };

export const CRONTrigger = ({ onChange, value }: CRONTriggerProps) => {
  const [error, setError] = useState("");
  const [val, setVal] = useState(value || "");
  const [readable, setReadable] = useState("");

  useEffect(() => {
    if (val) {
      try {
        setReadable(cronstrue.toString(val));
      } catch (e) {
        setError("Invalid cron schedule");
      }
    }
  }, [val]);

  const onValueChange = (e: any) => {
    setReadable("");
    setError("");
    setVal(e.target.value);
    onChange(e.target.value);
  };

  return (
    <CronContainer>
      <h3>Enter your CRON expression</h3>
      <div style={{ minHeight: "80px" }}>
        <input
          defaultValue={value}
          placeholder="e.g. '* * * * *'"
          onChange={onValueChange}
        />
        {error && <div style={{ color: "red" }}>{error}</div>}
        {readable && <div style={{ color: "gray" }}>{readable}</div>}
      </div>
      <CRONExamples>
        <p>
          <b>Common expressions:</b>
        </p>
        <small>
          <div>Every 15 minutes</div>
          <code>*/15 * * * *</code>
          <div>Every hour</div>
          <code>0 * * * *</code>
          <div>Every twelve hours</div>
          <code>0 */12 * * *</code>
          <div>Every day at midnight - 12am UTC</div>
          <code>0 0 * * *</code>
          <div>
            See more examples{" "}
            <a
              target="_blank"
              rel="noopener noreferrer"
              href="https://www.freeformatter.com/cron-expression-generator-quartz.html"
            >
              here
            </a>
            .
          </div>
        </small>
      </CRONExamples>
    </CronContainer>
  );
};

const EventTrigger = ({
  onChange,
  value,
}: {
  onChange: (event: string) => void;
  value: string;
}) => {
  const [eventSearch, setEventSearch] = useState<string>("");
  const [evtName] = useState<string>("");

  const [{ data }] = useEventNames(eventSearch);
  const [evtResult, evtFetch] = useEventDetails([evtName]);

  // memoize sorting names
  const results = useMemo(() => {
    const searchResults = (data ? data.workspace.events.data : []).map(
      (e: any) => {
        return {
          value: e.name,
          data: e,
          render: ({ className, onClick }: ResultProps) => {
            return (
              <div key={e.name} className={className} onClick={onClick}>
                <span>
                  <EventIcon />
                  <b>{e.name}</b>
                </span>
              </div>
            );
          },
        };
      }
    );

    if (searchResults.length) return searchResults;

    // if no existing events match, offer to create new event based on input
    return [
      {
        value: eventSearch,
        data: { name: eventSearch },
        render: ({ className, onClick }: ResultProps) => {
          return (
            <div className={className} onClick={onClick}>
              <span>
                <b>Use unreceived event "{eventSearch}"</b>
              </span>
            </div>
          );
        },
      },
    ];
  }, [data, eventSearch]);

  // on select, get broader event details
  useEffect(() => {
    if (!evtName) return;
    evtFetch({ variables: { names: [evtName] } });
  }, [evtName, evtFetch]);

  return (
    <div style={{ alignSelf: "flex-start" }}>
      <label>
        <h3>What event triggers this workflow?</h3>
        <Search
          placeholder="Search events..."
          defaultValue={value}
          onChange={setEventSearch}
          onSelect={(result) => {
            setEventSearch(result.data.name);
            onChange(result.data.name);
          }}
          onBlur={() => onChange(eventSearch)}
          icon={false}
          results={results}
          css={css`
            border: 1px solid #eee;
            margin-top: 10px;
            min-width: 620px;
          `}
        />
      </label>
      <EventDetailsMapper
        events={evtResult.data ? evtResult.data.workspace.events : []}
      />
    </div>
  );
};

export default NewWorkflowPage;

const EventDetailsMapper = ({ events }: { events: Event[] }) => {
  if (events.length === 0) return null;

  return (
    <div>
      {events.map((e) => (
        <div key={e.name}>
          <EventDetails event={e} />
          <hr />
        </div>
      ))}
    </div>
  );
};

const EventDetails = ({ event }: { event: Event }) => (
  <Data>
    <h4>Version</h4>
    <p>{event.version || "(no version)"}</p>

    <h4>Fields</h4>
    <ShapeDisplay shape={event.fields} />
  </Data>
);

const Wrapper = styled.div`
  position: absolute;
  display: flex;
  flex-flow: column;

  height: 100%;
  width: 100%;
`;

const Data = styled.div`
  display: flex;
  flex-direction: column;

  h4 {
    text-transform: uppercase;
    letter-spacing: 1px;
    opacitu: .5;
    font-size: .7rem;
    margin: 0 0 0.2;5rem;
  }

  p {
    font-size: .8rem;
  }

  p + h4 {
    margin-top: 1rem;
  }

  code {
    font-size: .8rem;
  }
`;

const Content = styled.div`
  z-index: 20;
  // background: rgba(240, 240, 240, 0.3);
  padding: 60px 40px;

  display: flex;
  flex-direction: column;
  align-items: stretch;

  h2 {
    margin: 0 0 30px;
  }

  label {
    margin: 20px 0;
    align-self: stretch;
    width: 680px;
  }

  flex: 1;
  overflow-y: auto;
`;

const Header = styled.div`
  padding: 10px;
  background: #fff;
  border-bottom: 1px solid #eee;
  height: 60px;
  display: flex;
  align-items: center;

  z-index: 6;
`;
