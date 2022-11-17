import { ApolloProvider } from "@apollo/client";
import { render } from "preact";
import { App } from "./app";
import { client } from "./coreapi";
import "./index.css";

render(
  <ApolloProvider client={client}>
    <App />
  </ApolloProvider>,
  document.getElementById("app") as HTMLElement
);
