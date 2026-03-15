import React from "react";

/* eslint-disable @typescript-eslint/no-explicit-any */

// Minimal Framework7 React mock for testing
export const f7 = {
  dialog: {
    alert: (..._args: unknown[]) => {},
    confirm: (..._args: unknown[]) => {},
  },
  toast: {
    create: () => ({ open: () => {} }),
  },
};

export const App = ({ children }: any) => <div>{children}</div>;
export const Page = ({ children }: any) => <div data-testid="page">{children}</div>;
export const Navbar = ({ title, backLink }: any) => (
  <nav data-testid="navbar">
    {backLink && <span>{backLink}</span>}
    <span>{title}</span>
  </nav>
);
export const List = ({ children }: any) => <ul>{children}</ul>;
export const ListInput = ({ label, type, value, onInput, children, placeholder, disabled }: any) => (
  <label>
    {label}
    {type === "select" ? (
      <select value={value} onChange={onInput} aria-label={label} disabled={disabled}>
        {children}
      </select>
    ) : type === "textarea" ? (
      <textarea value={value} onChange={onInput} placeholder={placeholder} aria-label={label} disabled={disabled} />
    ) : (
      <input
        type={type ?? "text"}
        value={value}
        onChange={onInput}
        placeholder={placeholder}
        aria-label={label}
        disabled={disabled}
      />
    )}
  </label>
);
export const ListItem = ({ title, subtitle, after, children, onClick }: any) => (
  <li onClick={onClick}>
    <span>{title}</span>
    {subtitle && <span>{subtitle}</span>}
    {after && <span>{after}</span>}
    {children}
  </li>
);
export const Button = ({ children, onClick, disabled, fill }: any) => (
  <button onClick={onClick} disabled={disabled} data-fill={fill ? "true" : "false"}>
    {children}
  </button>
);
export const BlockTitle = ({ children }: any) => <h3>{children}</h3>;
export const Block = ({ children, style }: any) => <div style={style}>{children}</div>;
export const Card = ({ children }: any) => <div data-testid="card">{children}</div>;
export const CardHeader = ({ children }: any) => <div>{children}</div>;
export const CardContent = ({ children }: any) => <div>{children}</div>;
export const Link = ({ children, onClick }: any) => (
  <a onClick={onClick} href="#">
    {children}
  </a>
);
export const Chip = ({ text, onClick, outline }: any) => (
  <span onClick={onClick} data-outline={outline ? "true" : "false"}>
    {text}
  </span>
);
export const Views = ({ children }: any) => <div>{children}</div>;
export const View = ({ children }: any) => <div>{children}</div>;
export const Toolbar = ({ children }: any) => <div>{children}</div>;
export const SwipeoutActions = ({ children }: any) => <div>{children}</div>;
export const SwipeoutButton = ({ children, onClick }: any) => (
  <button onClick={onClick}>{children}</button>
);
