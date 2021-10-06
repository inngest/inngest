import React, { useEffect } from "react";
import styled from "@emotion/styled";
// import { Filter } from "react-feather";
import { categoryIcons, defaultIcon } from "./icons";
import { useActionsWithCategory } from "./query";
import { Action } from "src/types";

type Props = {
  // Preloaded actions.  If not specified, this component will query for actions.
  actions?: Action[];
  // disabled represents whether the user is allowed to interact with the list
  disabled?: boolean;
  onDragStart?: (a: Action) => void;
};

// ActionPanel handles loading
const ActionPanel: React.FC<Props> = (props) => {
  const [{ fetching, error, data }, exec] = useActionsWithCategory(true);

  useEffect(() => {
    if (!props.actions) {
      exec();
    }
  }, []);

  if ((!props.actions && !data) || fetching || error) {
    return null;
  }

  const actions = props.actions || (data ? data.actions : []);

  return (
    <div>
      {/*
      <FilterButton>
        <p>Filter by: all</p>
        <Filter size={12} />
      </FilterButton>
      */}

      {actions.map((a) => (
        <ActionItem action={a} key={a.dsn} onDragStart={props.onDragStart} />
      ))}
    </div>
  );
};

const ActionItem: React.FC<{
  action: Action;
  onDragStart?: (a: Action) => void;
}> = ({ action, onDragStart }) => {
  const onDrag = (e: React.DragEvent<HTMLDivElement>) => {
    (e.dataTransfer as DataTransfer).effectAllowed = "copy";
    (e.dataTransfer as DataTransfer).setData("text", JSON.stringify(action));
    onDragStart && onDragStart(action);
  };

  const Icon = categoryIcons[action.category.name] || defaultIcon;

  return (
    <Wrapper draggable onDragStart={onDrag}>
      <div>
        <Icon />
      </div>
      <div>
        <p>{action.latest.name}</p>
        <small>{action.tagline}</small>
      </div>
    </Wrapper>
  );
};

const Wrapper = styled.div`
  display: grid;
  grid-template-columns: 40px auto;
  align-items: center;
  padding: 20px;
  box-shadow: 0;
  transition: all 0.2s;

  border-top: 1px solid #f5f5f5;

  &:last-of-type {
    border-bottom: 1px solid #f5f5f5;
  }

  svg {
    opacity: 0.6;
    margin: 0 10px 0 0;
  }

  p {
    display: flex;
    flex-direction: row;
    align-items: center;
  }

  &:hover {
    background: #fff;
    cursor: move;
    box-shadow: 0 5px 40px rgba(0, 0, 0, 0.12), 0 1px 4px rgba(0, 0, 0, 0.05);
  }
`;

const FilterButton = styled.button`
  display: flex;
  align-items: center;
  justify-content: space-between;
  border: 0;
  background: #fff;
  font-size: 12px;

  width: calc(100% - 30px);
  margin: 0 10px 5px;
  padding: 10px;
  opacity: 0.5;

  border: 1px solid #fff;
  border-radius: 3px;

  transition: all 0.2s;
  &:hover {
    border: 1px solid #eee;
    opacity: 1;
  }
`;

export default ActionPanel;
