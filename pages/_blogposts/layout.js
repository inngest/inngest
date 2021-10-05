import Footer from "../../shared/footer";
import Nav from "../../shared/nav";
import Content from "../../shared/content";
import { Wrapper, Inner } from "../../shared/blog";

export default function BlogLayout() {
  return (
    <Wrapper>
      <Nav />
      <Content>
        <Inner>$1</Inner>
      </Content>
      <Footer />
    </Wrapper>
  );
}
