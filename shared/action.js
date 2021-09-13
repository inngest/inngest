import styled from "@emotion/styled";

export default function Action({ name, subtitle, icon }) {
  return (
    <Wrapper>
      <img src={icon} />
      <div>
        <p>{name}</p>
        <small>{subtitle}</small>
      </div>
    </Wrapper>
  )
}

export const Outline = styled.div`
  border-radius: 5px;
  height: 80px;
  width: 280px;
  border: 4px dashed #e8e8e6;
  box-sizing: border-box;
`;

const Wrapper = styled.div`
  box-sizing: border-box;
  border-radius: 5px;
  background: #fff;
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.05);
  height: 80px;
  width: 280px;
  padding: 0 20px;
  border: 1px solid #e8e8e6;
  position: relative;
  z-index: 1;

  display: grid;
  grid-template-columns: 55px auto;

  align-items: center;

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
