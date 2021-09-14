import styled from "@emotion/styled";
import Tag from "./tag";

export default function Action({ name, subtitle, icon, className }) {
  return (
    <Wrapper className={`action ${className || ""}`}>
      <img src={icon} />
      <div>
        <p>{name}</p>
        <small>{subtitle}</small>
      </div>
    </Wrapper>
  );
}


export function Trigger({ name, icon, className }) {
  return (
    <TriggerWrapper className={`action trigger ${className || ""}`}>
      <Tag>EVENT TRIGGER</Tag>
      <img src={icon} />
      <div>
        <p>{name}</p>
        <small><b>Whenever</b> this event is received</small>
      </div>
    </TriggerWrapper>
  );
}


export function If({ expression, className }) {
  return (
    <IfWrapper className={`if expression ${className || ""}`}>
      <pre><code>{expression}</code></pre>
      <small>If this is true continue</small>
    </IfWrapper>
  );
}

export const Outline = styled.div`
  border-radius: 5px;
  height: 80px;
  width: 280px;
  border: 4px dashed #e8e8e6;
  box-sizing: border-box;
`;

export const Empty = styled.div`
  border-radius: 5px;
  min-height: 60px;
  width: 280px;
  box-sizing: border-box;
`;

export const IfWrapper = styled.div`
  position: relative;
  border-radius: 5px;
  height: 80px;
  width: 280px;
  box-sizing: border-box;
  background: rgb(246, 246, 246);
  border: 1px solid #e8e8e6;
  font-size: 12px;
  line-height: 1.2;
  color: #303e2fcc;
  display: flex;

  flex-direction: column;
  align-items: center;
  justify-content: center;

  pre { margin: 0 0 5px; }
  small { display: block; }
`;

const Wrapper = styled.div`
  box-sizing: border-box;
  border-radius: 5px;
  background: #fff;
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.05);
  height: 80px;
  width: 280px;
  padding: 0 15px 0 20px;
  border: 1px solid #e8e8e6;
  position: relative;
  z-index: 1;
  color: rgb(16, 63, 16);
  line-height: 0.7;

  display: grid;
  grid-template-columns: 50px auto;
  align-items: center;
  text-align: left;

  > span {
    position: absolute;
    top: -10px;
    left: calc(50% - 56px);
    z-index: 6;
  }

  > svg {
    align-self: center;
    justify-self: center;
    opacity: 0.5;
  }

  p {
    line-height: 1.2;
    margin-bottom: 1px;
    font-size: 14px;
    font-weight: 500;
    margin: 0 0 1px;
  }

  small {
    font-size: 11px;
    opacity: 0.6;
  }
`;

const TriggerWrapper = styled(Wrapper)`
  padding-top: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgb(206, 236, 206);
  text-align: center;

  p {
    margin: 0px;
    font-weight: 500;
  }

  small {
    font-size: 12px;
    opacity: 0.6;
    line-height: 1.1;
  }
`;

