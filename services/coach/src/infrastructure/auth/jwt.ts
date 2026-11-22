import { readFileSync } from "node:fs";
import { importSPKI, jwtVerify, type CryptoKey } from "jose";
import type { Principal, Role } from "../../domain/principal.js";

export interface Authenticator {
  authenticate(authorizationHeader: string | undefined): Promise<Principal | null>;
}

/** Same contract as the API (ADR 0011): RS256, iss, aud, exp, sub as UUID. The role comes from the `roles` claim (ADR 0014); a token with both roles acts as the athlete. */
export class JwtAuthenticator implements Authenticator {
  private key: Promise<CryptoKey> | undefined;
  constructor(private readonly opts: { publicKeyFile: string; issuer: string; audience: string }) {}

  async authenticate(header: string | undefined): Promise<Principal | null> {
    const token = /^Bearer (.+)$/i.exec(header ?? "")?.[1];
    if (!token) return null;
    try {
      this.key ??= importSPKI(readFileSync(this.opts.publicKeyFile, "utf8"), "RS256");
      const { payload } = await jwtVerify(token, await this.key, {
        issuer: this.opts.issuer,
        audience: this.opts.audience,
        algorithms: ["RS256"],
        requiredClaims: ["exp", "sub"],
      });
      const sub = String(payload.sub);
      if (!/^[0-9a-f-]{36}$/i.test(sub)) return null;
      // ADR 0014: `roles` lists atleta/treinador/admin; absent means atleta. Admin never reads health data.
      const roles: unknown[] = Array.isArray(payload.roles) ? payload.roles : ["atleta"];
      const isAthlete = roles.includes("atleta");
      if (!isAthlete && !roles.includes("treinador")) return null;
      const role: Role = isAthlete ? "athlete" : "coach";
      return { subject: sub, role, token };
    } catch {
      return null;
    }
  }
}
