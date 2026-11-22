/**
 * Free text from imported files (activity title, notes) is data. It is trimmed, stripped of control
 * and bidi characters, and later wrapped as a JSON string value; the prompt never concatenates it
 * into instructions. Defence does not rely on the prompt: the model's output is schema-validated and
 * cannot name anything outside the plan (ADR 0023).
 */
const MAX_UNTRUSTED = 120;

// eslint-disable-next-line no-control-regex, no-irregular-whitespace
const CONTROL = /[\u0000-\u001f\u007f-\u009f​-‏‪-‮⁦-⁩]/g;

export function sanitizeImportedText(raw: string | null | undefined): string {
  if (!raw) return "";
  return raw.replace(CONTROL, " ").replace(/[`<>{}]/g, "").replace(/\s+/g, " ").trim().slice(0, MAX_UNTRUSTED);
}
