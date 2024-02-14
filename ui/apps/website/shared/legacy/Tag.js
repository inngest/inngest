import styled from "@emotion/styled";

export default function (props) {
  return (
    <Wrapper {...props} className={`rounded-sm ${props.className || ""}`}>
      {props.children}
    </Wrapper>
  );
}

const Wrapper = styled.div`
  display: inline-block;
  background: var(--color-light-purple);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 1.5px;
  font-weight: bold;
  padding: 0.25rem 0.5rem;
`;
