import React from "react";
import ReactDOM from "react-dom/client";
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
ReactDOM.createRoot(document.getElementById("app") as HTMLElement).render(
  <React.StrictMode>
    <Provider store={store}>
      <App />
    </Provider>
  </React.StrictMode>
);
