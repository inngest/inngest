import React from "react";
import styled from "@emotion/styled";

type Props = {
  href: string;
  children: React.ReactNode;
};

const PageBanner = styled.a<Props>`
  display: block;
  padding: 0.75em;
  background: #7e4ff5 linear-gradient(270deg, #4636f5 0%, #b565f3 100%);
  background-size: 100%;
  font-size: 14px;
  line-height: 1.5em; // 21px
  font-family: var(--font);
  text-align: center;
  color: #fff;
  text-decoration: none;
  transition: all 0.3s ease-in-out;

  &:hover {
    background-size: 150%;
  }
`;

export default PageBanner;
