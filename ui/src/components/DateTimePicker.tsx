import React from "react";
import { ListInput } from "framework7-react";

interface DateTimePickerProps {
  value: Date;
  onChange: (date: Date) => void;
  label?: string;
}

export const DateTimePicker: React.FC<DateTimePickerProps> = ({
  value,
  onChange,
  label = "Date & Time",
}) => {
  const formatted = formatForInput(value);

  return (
    <ListInput
      label={label}
      type="datetime-local"
      value={formatted}
      onInput={(e: React.ChangeEvent<HTMLInputElement>) => {
        const val = e.target.value;
        if (val) {
          onChange(new Date(val));
        }
      }}
    />
  );
};

function formatForInput(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}
