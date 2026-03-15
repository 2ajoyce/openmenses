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
  return (
    <div className="enum-selector">
      {label && (
        <div className="enum-selector-label block-title">{label}</div>
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
          >
            {opt.label}
          </Button>
        ))}
      </div>
    </div>
  );
};
