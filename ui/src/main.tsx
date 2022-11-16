import { ApolloClient, ApolloProvider, InMemoryCache } from "@apollo/client";
import { render } from "preact";
import { App } from "./app";
import "./index.css";

const client = new ApolloClient({
  uri: "http://localhost:4000/graphql",
  cache: new InMemoryCache(),
});

render(
  <ApolloProvider client={client}>
    <App />
  </ApolloProvider>,
  document.getElementById("app") as HTMLElement
);
