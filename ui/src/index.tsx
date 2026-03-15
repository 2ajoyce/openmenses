import Framework7 from "framework7";
import "framework7-icons/css/framework7-icons.css";
import Framework7React from "framework7-react";
import "framework7/css/bundle";
import { createRoot } from "react-dom/client";
import App from "./app/App";

Framework7.use(Framework7React);

const root = createRoot(document.getElementById("app")!);
root.render(<App />);
