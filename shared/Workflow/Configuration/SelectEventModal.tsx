import React from "react";
import styled from "@emotion/styled";
import { css } from "@emotion/react";
import { DateTime } from "luxon";
import { useWorkflowContext } from "../state";
import Modal from "src/shared/Modal";
import Box from "src/shared/Box";
import Table from "src/shared/Table";
import { gqlError } from "src/utils";
import { displayName } from "src/utils/contact";
import { useCurrentWorkspace } from "src/state/workspaces";
import { useEventRecents, RecentEvent } from "../queries";

const SelectEventModal: React.FC<{ onClose: () => void }> = (props) => {
  const w = useCurrentWorkspace();
  const [state, dispatch] = useWorkflowContext();

  // XXX: This should work with > 1 trigger.
  const eventName = React.useMemo((): string => {
    if (!state.workflow) return "";
    const evt = state.workflow.triggers.find((t) => !!t.event);
    return evt && evt.event ? evt.event : "";
  }, [state.workflow]);

  const [{ fetching, error, data }] = useEventRecents(w.id, eventName, 25);

  return (
    <Modal onClose={props.onClose}>
      <ModalBox>
        <h2>Select an event</h2>
        <p>
          Select an event to use as an example for this workflow.{" "}
          <span style={{ opacity: 0.5 }}>Showing the latest 25 events.</span>
        </p>

        <Table<RecentEvent>
          columns={["Contact", "Data", "ID", "Occurred", "Occurred time"]}
          columnCSS={css`
            grid-template-columns: 200px auto 210px 120px 200px;
          `}
          loading={fetching}
          error={error && gqlError(error)}
          data={data && data.workspace.event ? data.workspace.event.recent : []}
          row={({ item, className }) => (
            <a
              href="#"
              className={className}
              onClick={() => {
                dispatch({ type: "exampleEvent", event: item });
                props.onClose();
              }}
            >
              <div>
                {item.contact ? (
                  <b>{displayName(item.contact.predefinedAttributes)}</b>
                ) : (
                  <p style={{ opacity: 0.5 }}>No contact</p>
                )}
              </div>
              <EventData>{item.event}</EventData>
              <EventData>{item.id}</EventData>
              <div>{DateTime.fromISO(item.occurredAt).toRelative()}</div>
              <div>
                {DateTime.fromISO(item.occurredAt).toLocaleString(
                  DateTime.DATETIME_SHORT
                )}
              </div>
            </a>
          )}
          blank={() => (
            <div>
              <p>
                No events with the name <code>{eventName}</code> stored.
              </p>
            </div>
          )}
          keyFn={(e) => e.id}
        />
      </ModalBox>
    </Modal>
  );
};

const ModalBox = styled(Box)`
  min-width: 50vw;
  max-width: 80vw;
  max-height: 80vh;
  overflow-y: scroll;
  padding-bottom: 60px;
  padding: 30px;

  .table {
    margin-top: 30px;
  }

  /* hack for padding bottom while scroll is still on this parent - no floating scrollbar */
  &:after {
    content: "";
    display: block;
    background: #fff;
    position: absolute;
    left: 0;
    bottom: 0;
    width: 100%;
    height: 30px;
  }
`;

const EventData = styled.div`
  text-overflow: ellipsis;
  overflow: hidden;
  white-space: nowrap;
  font-family: monospace;
  font-size: 12px;
`;

export default SelectEventModal;
