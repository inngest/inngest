import { render } from "preact";
import { Provider } from "react-redux";
import { App } from "./app";
import "./index.css";
import { store } from "./store/store";

/**
 * We're using Preact rather than React here, so the `react-redux` `Provider`
 * gets a bit confused about what we're passing it.
 *
 * Let's avoid the issue here.
 */
const P = Provider as any;

render(
  <P store={store}>
    <App />
  </P>,
  document.getElementById("app") as HTMLElement
);
