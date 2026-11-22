import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import Fastify, { type FastifyInstance } from "fastify";
import { z } from "zod";
import type { ApprovalService } from "../../application/approval.js";
import { CoachError } from "../../application/errors.js";
import { resolveSubject, type CoachService } from "../../application/service.js";
import type { Principal } from "../../domain/principal.js";
import type { Authenticator } from "../../infrastructure/auth/jwt.js";
import { createCoachMcpServer } from "../mcp/server.js";

const STATUS: Record<CoachError["code"], number> = {
  forbidden: 403, not_found: 404, conflict: 409, budget_exceeded: 429, upstream: 502, invalid: 400,
};

const MessageBody = z.object({
  intent: z.enum(["adjust_week", "explain_session", "weekly_summary", "parse_goal"]),
  message: z.string().min(1).max(1000),
  session_id: z.string().min(1).max(100).optional(),
  athlete_id: z.uuid().optional(),
});

declare module "fastify" {
  interface FastifyRequest {
    principal?: Principal;
  }
}

export function buildApp(deps: { auth: Authenticator; service: CoachService; approvals: ApprovalService }): FastifyInstance {
  const app = Fastify({ logger: false });

  app.setErrorHandler((err, _req, reply) => {
    if (err instanceof CoachError) return reply.code(STATUS[err.code]).send({ error: err.code, message: err.message });
    if (err instanceof z.ZodError) return reply.code(400).send({ error: "invalid", message: "invalid request body" });
    return reply.code(500).send({ error: "internal", message: "internal error" });
  });

  app.get("/health", () => ({ status: "ok" }));

  app.addHook("preHandler", async (req, reply) => {
    if (req.url === "/health") return;
    const principal = await deps.auth.authenticate(req.headers.authorization);
    if (!principal) return reply.code(401).send({ error: "unauthorized", message: "invalid or missing token" });
    req.principal = principal;
  });

  app.post("/coach/messages", async (req) => {
    const body = MessageBody.parse(req.body);
    const subject = resolveSubject(req.principal!, body.athlete_id);
    return deps.service.ask(subject, { intent: body.intent, message: body.message, sessionId: body.session_id });
  });

  app.get("/coach/proposals", async (req) => {
    const q = z.object({ athlete_id: z.uuid().optional() }).parse(req.query);
    const subject = resolveSubject(req.principal!, q.athlete_id);
    return { items: await deps.approvals.list(subject.principal, subject.athleteId) };
  });

  const IdParam = z.object({ id: z.uuid() });
  app.post("/coach/proposals/:id/approve", async (req) =>
    deps.approvals.approve(req.principal!, IdParam.parse(req.params).id));
  app.post("/coach/proposals/:id/reject", async (req) =>
    deps.approvals.reject(req.principal!, IdParam.parse(req.params).id));

  // Stateless Streamable HTTP: one server and transport per request, bound to the caller's token.
  app.post("/mcp", async (req, reply) => {
    const athlete = z.uuid().optional().parse(req.headers["x-athlete-id"]);
    const subject = resolveSubject(req.principal!, athlete);
    const server = createCoachMcpServer(deps.service.toolkitFor(subject));
    const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined });
    reply.hijack();
    reply.raw.on("close", () => {
      void transport.close();
      void server.close();
    });
    await server.connect(transport);
    await transport.handleRequest(req.raw, reply.raw, req.body);
  });

  return app;
}
