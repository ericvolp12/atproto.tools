import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClientProvider } from "react-query";
import queryClient from "./queryClient";
import App from "./App.tsx";

import "./index.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>,
);
