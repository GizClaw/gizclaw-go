import { X } from "lucide-react";

import { Button } from "./button";
import { Input } from "./input";

type DateTimeInputProps = {
  disabled?: boolean;
  onChange: (value: string) => void;
  value: string;
};

export function DateTimeInput({ disabled = false, onChange, value }: DateTimeInputProps): JSX.Element {
  return (
    <div className="flex gap-2">
      <Input
        disabled={disabled}
        onChange={(event) => onChange(rfc3339FromLocalInput(event.target.value))}
        step={60}
        type="datetime-local"
        value={localInputFromRFC3339(value)}
      />
      <Button aria-label="Clear datetime" className="shrink-0" disabled={disabled || value.trim() === ""} onClick={() => onChange("")} size="icon" type="button" variant="outline">
        <X className="size-4" />
      </Button>
    </div>
  );
}

function localInputFromRFC3339(value: string): string {
  const trimmed = value.trim();
  if (trimmed === "") {
    return "";
  }
  const time = Date.parse(trimmed);
  if (Number.isNaN(time)) {
    return "";
  }
  const date = new Date(time);
  const year = String(date.getFullYear()).padStart(4, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hour = String(date.getHours()).padStart(2, "0");
  const minute = String(date.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hour}:${minute}`;
}

function rfc3339FromLocalInput(value: string): string {
  if (value.trim() === "") {
    return "";
  }
  const time = Date.parse(value);
  if (Number.isNaN(time)) {
    return "";
  }
  return new Date(time).toISOString().replace(/\.\d{3}Z$/, "Z");
}
