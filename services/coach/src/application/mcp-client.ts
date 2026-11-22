import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { InMemoryTransport } from "@modelcontextprotocol/sdk/inMemory.js";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";

export class ToolError extends Error {
  constructor(readonly tool: string, readonly body: string) {
    super(`tool ${tool} failed: ${body}`);
  }
}

/** Talks to the coach MCP server in-process through a linked transport pair, the same protocol an external client uses. */
export class McpToolClient {
  private constructor(private readonly client: Client, private readonly server: McpServer) {}

  static async connect(server: McpServer): Promise<McpToolClient> {
    const [a, b] = InMemoryTransport.createLinkedPair();
    const client = new Client({ name: "atp-coach-graph", version: "0.1.0" });
    await Promise.all([server.connect(a), client.connect(b)]);
    return new McpToolClient(client, server);
  }

  async call<T = unknown>(name: string, args: Record<string, unknown> = {}): Promise<T> {
    const res = await this.client.callTool({ name, arguments: args });
    const first = (res.content as Array<{ type: string; text?: string }>)[0];
    const text = first?.text ?? "null";
    if (res.isError) throw new ToolError(name, text);
    return JSON.parse(text) as T;
  }

  async close(): Promise<void> {
    await this.client.close();
    await this.server.close();
  }
}
