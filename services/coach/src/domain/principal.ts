export type Role = "athlete" | "coach";

/** Who is calling. The token is kept only to forward it to the HTTP ports; it never reaches the LLM. */
export interface Principal {
  readonly subject: string;
  readonly role: Role;
  readonly token: string;
}

/** The athlete whose data a request is about. For an athlete it is the caller; for a coach it comes from the request. */
export interface Subject {
  readonly principal: Principal;
  readonly athleteId: string;
}
