import React from "react";
import { Block, Button, Icon } from "framework7-react";

interface EmptyStateProps {
  message: string;
  actionLabel?: string;
  onAction?: () => void;
  icon?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  message,
  actionLabel,
  onAction,
  icon,
}) => {
  return (
    <Block className="om-empty-state">
      {icon && (
        <div className="om-empty-state-icon">
          <Icon f7={icon} />
        </div>
      )}
      <p className="om-empty-state-message">{message}</p>
      {actionLabel && onAction && (
        <Button fill round onClick={onAction}>
          {actionLabel}
        </Button>
      )}
    </Block>
  );
};
