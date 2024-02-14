import React from "react";
import styled from "@emotion/styled";

const SectionHeader: React.FC<{
  title: string | React.ReactNode;
  subtitle: string;
  className?: string;
  size?: "large" | "default";
  align?: "left" | "center";
}> = ({ title, subtitle, className, size = "default", align = "center" }) => {
  return (
    <Wrapper align={align} className={className || ""}>
      <Heading size={size}>{title}</Heading>
      <Subheading align={align}>{subtitle}</Subheading>
    </Wrapper>
  );
};

const Wrapper = styled.div<{ align: "left" | "center" }>`
  max-width: var(--max-page-width);
  margin: 3rem auto;
  display: flex;
  flex-direction: column;
  align-items: ${({ align }) => (align === "center" ? "center" : "flex-start")};
  text-align: ${({ align }) => align};

  @media (max-width: 1240px) {
    margin-left: 2rem;
    margin-right: 2rem;
  }

  @media (max-width: 700px) {
    margin-bottom: 2rem;
    font-size: 0.8rem;
  }
`;

const Heading = styled.h2<{ size: "large" | "default" }>`
  margin: 0 0 0.2em;
  max-width: 45rem;
  font-size: ${({ size }) => (size === "large" ? "2em" : "1.8em")};
`;
const Subheading = styled.p<{ align: "left" | "center" }>`
  margin: 0.2rem 0 0;
  max-width: ${({ align }) => (align === "center" ? "30rem" : "100%")};
  font-size: 1.1em;
  color: var(--font-color-secondary);
`;

export default SectionHeader;
