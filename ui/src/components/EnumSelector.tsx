import React from "react";
import { Button } from "framework7-react";

interface EnumOption {
  value: number;
  label: string;
}

interface EnumSelectorProps {
  options: EnumOption[];
  selected: number;
  onChange: (value: number) => void;
  label?: string;
}

export const EnumSelector: React.FC<EnumSelectorProps> = ({
  options,
  selected,
  onChange,
  label,
}) => {
  const labelId = label ? `enum-selector-${label.replace(/\s+/g, "-").toLowerCase()}` : undefined;

  return (
    <div className="enum-selector" role="group" aria-labelledby={labelId}>
      {label && (
        <div id={labelId} className="enum-selector-label block-title">{label}</div>
      )}
      <div className="enum-selector-options">
        {options.map((opt) => (
          <Button
            key={opt.value}
            fill={selected === opt.value}
            outline={selected !== opt.value}
            round
            small
            onClick={() => onChange(opt.value)}
            aria-pressed={selected === opt.value}
            aria-label={opt.label}
          >
            {opt.label}
          </Button>
        ))}
      </div>
    </div>
  );
};
