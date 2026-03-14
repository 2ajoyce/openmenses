import { create } from "@bufbuild/protobuf";
import { LocalDateSchema, DateTimeSchema } from "@gen/openmenses/v1/types_pb";
import type { LocalDate, DateTime } from "@gen/openmenses/v1/types_pb";

export function toLocalDate(date: Date): LocalDate {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return create(LocalDateSchema, { value: `${year}-${month}-${day}` });
}

export function toDateTime(date: Date): DateTime {
  return create(DateTimeSchema, { value: date.toISOString().replace(/\.\d{3}Z$/, "Z") });
}

export function fromLocalDate(ld: LocalDate): Date {
  const [year, month, day] = ld.value.split("-").map(Number);
  return new Date(year!, month! - 1, day);
}

export function fromDateTime(dt: DateTime): Date {
  return new Date(dt.value);
}

export function formatDate(ld: LocalDate): string {
  const date = fromLocalDate(ld);
  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function formatTime(dt: DateTime): string {
  const date = fromDateTime(dt);
  return date.toLocaleTimeString("en-US", {
    hour: "numeric",
    minute: "2-digit",
  });
}

export function formatDateTime(dt: DateTime): string {
  const date = fromDateTime(dt);
  return date.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export function nowDateTime(): DateTime {
  return toDateTime(new Date());
}

export function daysAgo(n: number): Date {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d;
}
