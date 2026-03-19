import React from "react";
import { ListInput } from "framework7-react";

interface NotesFieldProps {
  value: string;
  onChange: (value: string) => void;
  maxLength?: number;
}

export const NotesField: React.FC<NotesFieldProps> = ({
  value,
  onChange,
  maxLength = 1024,
}) => {
  const infoId = "notes-char-count";

  return (
    <ListInput
      label="Notes"
      type="textarea"
      placeholder="Optional notes..."
      value={value}
      maxlength={maxLength}
      onInput={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
        onChange(e.target.value)
      }
      info={`${value.length}/${maxLength}`}
      aria-describedby={infoId}
    />
  );
};
