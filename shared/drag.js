import styled from "@emotion/styled";
import Action, { Outline } from "../shared/action";

export default function Drag({ name, subtitle, icon, cursor })  {
  return (
    <Wrapper role="img" aria-label="Dragging an action onto a workflow">
      { cursor && <img src="/icons/drag.svg" aria-role="presentation" alt="" /> }
      <Action 
        name={name}
        subtitle={subtitle}
        icon={icon}
      />
      { cursor && <Outline className="drop" /> }
    </Wrapper>
  )
}

const Wrapper = styled.div`
  position: relative;

  > img {
    width: 24px;
    height: 24px;
    position: absolute;
    z-index: 2;
    right: 6px;
    top: 7px;
    pointer-events: none;
  }

  .drop {
    position: absolute;
    top: 10px;
    left: 10px;
    z-index: 0;
  }
`;

