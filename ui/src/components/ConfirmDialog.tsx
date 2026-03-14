import React, { useEffect, useRef } from "react";
import { f7 } from "framework7-react";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open,
  title,
  message,
  onConfirm,
  onCancel,
}) => {
  const shownRef = useRef(false);

  useEffect(() => {
    if (open && !shownRef.current) {
      shownRef.current = true;
      f7.dialog.confirm(message, title, () => {
        shownRef.current = false;
        onConfirm();
      }, () => {
        shownRef.current = false;
        onCancel();
      });
    }
    if (!open) {
      shownRef.current = false;
    }
  }, [open, title, message, onConfirm, onCancel]);

  return null;
};
