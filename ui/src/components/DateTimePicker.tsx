import React, { useId } from "react";

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
  const id = useId();
  const formatted = formatForInput(value);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
      <label
        htmlFor={id}
        style={{
          fontSize: "12px",
          color: "var(--f7-label-text-color, #6b7280)",
          fontWeight: 500,
        }}
      >
        {label}
      </label>
      <input
        id={id}
        type="datetime-local"
        value={formatted}
        onChange={(e) => {
          const val = e.target.value;
          if (val) {
            onChange(new Date(val));
          }
        }}
        style={{
          border: "1px solid var(--f7-input-outline-border-color, #c8c7cc)",
          borderRadius: "8px",
          padding: "6px 10px",
          fontSize: "14px",
          background: "var(--f7-input-bg-color, #fff)",
          color: "var(--f7-input-text-color, inherit)",
        }}
      />
    </div>
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
