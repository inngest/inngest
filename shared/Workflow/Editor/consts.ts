import { ConnectionLineType } from "react-flow-renderer";
import { State } from "../state";
import { Position, FlowElement } from "react-flow-renderer";

export const nodeW = 280;
export const nodeH = 80;
// addW is actually rendered as 40px wide;  this adds padding to dagre for layouts.
// XXX: Remove hack
export const addW = 100;
export const addH = 80;
export const conditionalHeight = 50;
export const conditionalMargin = 20;
export const gridSize = 25;
export const nodeMarginX = gridSize * 2;
export const nodeMarginY = 50;

export const edgeType = "step" as ConnectionLineType;

export const getNodeHeight = (type: string) => {
  switch (type) {
    case "add":
      return addH;
    case "blankExpression":
      return conditionalHeight;
    case "expression":
      return conditionalHeight;
    case "conditionalAction":
      return nodeH + conditionalHeight;
    default:
      return nodeH;
  }
};

export const newClientID = (s: State): number => {
  if (!s.workflow || s.workflow.actions.length === 0) {
    return 1;
  }
  return (
    (s.workflow.actions
      .map((a) => a.clientID)
      .sort((a, b) => b - a)
      .shift() as number) +
    1 +
    s.addMutations.length +
    (s.dragMutation ? 1 : 0)
  );
};

export const newNode = ({
  id,
  type,
  data,
  position = { x: 0, y: 0 },
}: {
  id: string;
  type: string;
  data: Object;
  position?: { x: number; y: number };
}): FlowElement => ({
  id,
  type,
  sourcePosition: "bottom" as Position,
  targetPosition: "top" as Position,
  data,
  position, // updated by dagre
});

export const newEdge = ({
  source,
  target,
}: {
  source: string;
  target: string;
}): FlowElement => ({
  id: `edge-${source}-${target}`,
  type: edgeType,
  source,
  target,
  style: { stroke: "#E8E8E6", strokeWidth: 1 },
  position: { x: 0, y: 0 }, // updated by dagre
});

export const createAddNodes = (clientID: "trigger" | number | string) => {
  const items = [];
  items.push(
    newNode({
      id: clientID.toString() + "-add",
      type: "add",
      data: { outgoingID: clientID },
    })
  );
  items.push(
    newEdge({
      source: clientID.toString(),
      target: clientID.toString() + "-add",
    })
  );
  return items;
};
