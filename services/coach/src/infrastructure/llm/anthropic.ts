import Anthropic from "@anthropic-ai/sdk";
import { CoachError } from "../../application/errors.js";
import type { LlmPort, LlmRequest, LlmResponse } from "../../application/ports.js";

/** Real adapter. Not exercised against the live service in this repository (no credential); see the README. */
export class AnthropicLlm implements LlmPort {
  private readonly client: Anthropic;
  constructor(apiKey: string, private readonly model: string, timeoutMs: number) {
    this.client = new Anthropic({ apiKey, timeout: timeoutMs, maxRetries: 1 });
  }

  async complete(req: LlmRequest): Promise<LlmResponse> {
    try {
      const res = await this.client.messages.create({
        model: this.model,
        max_tokens: req.maxOutputTokens,
        system: req.system,
        messages: [{ role: "user", content: req.user }],
      });
      const text = res.content.map((b) => (b.type === "text" ? b.text : "")).join("");
      return { text, usage: { inputTokens: res.usage.input_tokens, outputTokens: res.usage.output_tokens } };
    } catch {
      throw new CoachError("upstream", "language model call failed");
    }
  }
}
