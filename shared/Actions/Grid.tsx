import React from "react";
import { Link } from "react-router-dom";
import styled from "@emotion/styled";
import { categoryIcons, defaultIcon } from "./icons";
import { Action } from "src/types";

type Props = {
  actions: Action[];
  onClick?: (a: Action) => void;
};

const ActionGrid: React.FC<Props> = (props) => {
  return (
    <Grid>
      {props.actions.map((a) => (
        <ActionItem action={a} key={a.dsn} onClick={props.onClick} />
      ))}
    </Grid>
  );
};

export default ActionGrid;

const ActionItem: React.FC<{
  action: Action;
  onClick?: (a: Action) => void;
}> = ({ action, onClick }) => {
  const Icon = categoryIcons[action.category.name] || defaultIcon;

  const Outer = onClick ? Div : Link;

  const onClickFn = () => onClick && onClick(action);

  return (
    <Outer
      to={`/workflows/actions/${encodeURIComponent(action.dsn)}`}
      style={{ color: "inherit", textDecoration: "none" }}
      onClick={onClickFn}
    >
      <Wrapper>
        <div>
          <Icon />
        </div>
        <div>
          <p>{action.latest.name}</p>
          <small>{action.tagline || "-"}</small>
        </div>
      </Wrapper>
    </Outer>
  );
};

const Grid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  grid-gap: 20px;
`;

const Div = styled.div``;

const Wrapper = styled.div`
  display: grid;
  grid-template-columns: 40px auto;
  align-items: center;
  padding: 20px;
  box-shadow: 0;
  transition: all 0.2s;

  border: 1px solid #f5f5f5;

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
    cursor: pointer;
    box-shadow: 0 5px 40px rgba(0, 0, 0, 0.12), 0 1px 4px rgba(0, 0, 0, 0.05);
  }
`;
