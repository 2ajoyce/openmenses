import React from "react";
import { Block, Button } from "framework7-react";

interface EmptyStateProps {
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  message,
  actionLabel,
  onAction,
}) => {
  return (
    <Block className="text-align-center" style={{ marginTop: "40px" }}>
      <p style={{ color: "var(--f7-text-color)", opacity: 0.55 }}>{message}</p>
      {actionLabel && onAction && (
        <Button fill round onClick={onAction} style={{ marginTop: "16px" }}>
          {actionLabel}
        </Button>
      )}
    </Block>
  );
};
