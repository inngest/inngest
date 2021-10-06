import { State, WorkflowEdge } from "./state";
import { Action as BaseAction } from "src/types";
import { Event, RecentEvent } from "./queries";

type DataObj = { [field: string]: string };

type ActionData = {
  clientID: number;
  dsn: string;
  name: string;
  data: DataObj;
};

export class AvailableData {
  event: EventData;
  actions: Array<ActionData>;
  user?: { [key: string]: any };

  example?: RecentEvent;

  // state is necessary to calculate available data from parent actions;  the
  // workflow state dictates which actions are available.
  state: State;

  constructor(s: State, fromActionID?: number, example?: RecentEvent) {
    this.state = s;
    this.event = new EventData(s.triggerEventDetails.filter(Boolean));
    this.actions = [];
    this.fromActionID(fromActionID);
    this.setExampleEvent(example);
  }

  // fromActionID sets the actions property to simulate data available from a specific
  // action ID, by looking up that action's parents.
  fromActionID = (fromActionID?: number) => {
    if (!fromActionID) {
      this.actions = [];
      return;
    }

    parents(this.state, fromActionID).forEach((item) => {
      const r = item.action.latest.Response || {};
      const result: DataObj = {};

      Object.keys(r).forEach((field) => {
        result[field] = r[field].type;
      });

      if (Object.keys(r).length === 0) {
        return;
      }

      this.actions.push({
        clientID: item.clientID,
        name: item.name,
        dsn: item.action.dsn,
        data: result,
      });
    });
  };

  // setExampleEvent uses an example event to show real-world data for a workflow.
  setExampleEvent = (example?: RecentEvent) => {
    this.example = example;

    if (!example) {
      this.event = new EventData(
        this.state.triggerEventDetails.filter(Boolean)
      );
      this.user = undefined;
      return;
    }
    this.event.setExampleEvent(example);

    // TODO: This may have user data, so fetch the user profile.
  };

  get displayJSON() {
    const action: any = {};
    this.actions.forEach((item) => {
      action[item.clientID] = item.data;
    });

    if (this.example && this.example.contact) {
      const user: any = {};
      this.example.contact.attributes.forEach((item) => {
        user[item.name] = JSON.parse(item.value);
      });

      return {
        event: this.event.displayJSON,
        action,
        user,
      };
    }

    return {
      event: this.event.displayJSON,
      action,
    };
  }
}

// showAvailableActionData shows the data which is available to a given action.
export const showAvailableActionData = (
  s: State,
  fromActionID?: number,
  exampleEvent?: RecentEvent
): AvailableData => {
  return new AvailableData(s, fromActionID, exampleEvent);
};

// EventData represents the available data from the event triggers.
class EventData {
  _multiple = false;
  _example: { [key: string]: any } | undefined = undefined;

  _events: {
    [eventName: string]: {
      name: string;
      data: DataObj;
      // TODO: User, which isn't exposed via the API yet.
    };
  } = {};

  constructor(e?: Event[]) {
    if (!e || e.length === 0) {
      return;
    }

    // TODO: Handle displaying multiple versions of the same event in a nice manner.
    e.forEach((e) => this.addEvent(e));
  }

  setExampleEvent = (e: RecentEvent) => {
    this._example = JSON.parse(e.event);
  };

  addEvent = (e: Event) => {
    if (!this._events[e.name]) {
      this._events[e.name] = { name: e.name, data: {} };
    }

    Object.keys(e.fields).forEach((key) => {
      if (e.fields[key].compound) {
        // TODO: Handle compound keys
      }

      this._events[e.name].data[key] = e.fields[key].scalar;
    });
  };

  get hasMultiple() {
    return this._multiple;
  }

  get displayJSON() {
    if (this._example) {
      return this._example;
    }

    const keys = Object.keys(this._events);
    if (keys.length === 1) {
      return this._events[keys[0]];
    }
    return this._events;
  }
}

const parents = (
  s: State,
  fromActionID: number
): Array<{ action: BaseAction; clientID: number; name: string }> => {
  const queue: Array<WorkflowEdge> = s.incomingActionEdges[
    fromActionID.toString()
  ].slice();

  const result: Array<{
    action: BaseAction;
    clientID: number;
    name: string;
  }> = [];

  while (queue.length > 0) {
    const item = queue.pop();
    const action = item && s.workflowActions[item.outgoing];
    if (!item || !action) {
      continue;
    }

    const base = s.actions.find((a) => a.dsn === action.dsn);
    base &&
      result.push({
        action: base,
        clientID: action.clientID,
        name: action.name,
      });

    // Add these parents to the queue.
    (s.incomingActionEdges[action.clientID.toString()] || []).forEach(
      (edge) => {
        queue.push(edge);
      }
    );
  }

  return result;
};
