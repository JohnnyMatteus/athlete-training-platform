export class CoachError extends Error {
  constructor(
    readonly code: "forbidden" | "not_found" | "upstream" | "budget_exceeded" | "conflict" | "invalid",
    message: string,
  ) {
    super(message);
  }
}
export const forbidden = (m: string) => new CoachError("forbidden", m);
export const notFound = (m: string) => new CoachError("not_found", m);
export const conflict = (m: string) => new CoachError("conflict", m);
