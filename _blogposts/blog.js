import styled from "@emotion/styled";
import Footer from "../shared/footer";
import Nav from "../shared/nav";
import Content from "../shared/content";
import { Wrapper } from "../shared/blog";

export default function BlogLayout() {
  return (
    <>
      <Wrapper>
        <Nav />
        <Content>
          <List>$1</List>
        </Content>
        <Footer />
      </Wrapper>
    </>
  );
}

const List = styled.div`
  box-sizing: border-box;
  padding: 100px 0;
  min-height: calc(100vh - 275px);

  h1 {
    margin: 0 0 2rem;
  }

  h2 {
    margin: 1rem 0 1rem;
    font-weight: 600;
    font-size: 1.65rem;
  }

  a {
    text-decoration: none;
    color: inherit;
  }

  .post--item {
    display: block;
    padding: 2rem;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.05);
    background: #fff;
    border-radius: 10px;
  }
`;
