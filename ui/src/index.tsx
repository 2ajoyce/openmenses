import "framework7/css/bundle";
import Framework7 from "framework7";
import Framework7React from "framework7-react";
import React from "react";
import { createRoot } from "react-dom/client";
import App from "./app/App";

Framework7.use(Framework7React);

const root = createRoot(document.getElementById("app")!);
root.render(<App />);
