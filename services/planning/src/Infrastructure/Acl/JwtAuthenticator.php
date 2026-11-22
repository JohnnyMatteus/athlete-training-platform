<?php

declare(strict_types=1);

namespace App\Infrastructure\Acl;

use App\Application\Port\Authenticator;
use App\Application\Unauthenticated;
use App\Domain\Shared\Principal;
use Firebase\JWT\JWT;
use Firebase\JWT\Key;
use Ramsey\Uuid\Uuid;
use Throwable;

/**
 * ACL do provedor de identidade (ADR 0011, 0014): valida o JWT RS256 e traduz `sub` e `roles` em Principal.
 * Nenhuma classe da biblioteca de JWT atravessa a porta.
 */
final readonly class JwtAuthenticator implements Authenticator
{
    private const KNOWN_ROLES = ['atleta', 'treinador', 'admin'];

    public function __construct(
        private string $publicKey,
        private string $issuer,
        private string $audience,
        private string $algorithm = 'RS256',
        private int $leewaySeconds = 30,
    ) {
    }

    public function authenticate(string $bearerToken): Principal
    {
        JWT::$leeway = $this->leewaySeconds;

        try {
            $claims = (array) JWT::decode($bearerToken, new Key($this->publicKey, $this->algorithm));
            if (($claims['iss'] ?? null) !== $this->issuer || !in_array($this->audience, (array) ($claims['aud'] ?? []), true)) {
                throw new Unauthenticated('emissor ou audiência inválidos');
            }
            if (!isset($claims['exp'])) {
                throw new Unauthenticated('token sem expiração');
            }
            $sub = (string) ($claims['sub'] ?? '');
            if (!Uuid::isValid($sub)) {
                throw new Unauthenticated('sub inválido');
            }
            $roles = isset($claims['roles']) ? array_values(array_intersect((array) $claims['roles'], self::KNOWN_ROLES)) : ['atleta'];

            return new Principal(strtolower($sub), $roles);
        } catch (Unauthenticated $e) {
            throw $e;
        } catch (Throwable $e) {
            throw new Unauthenticated('token inválido', 0, $e);
        }
    }
}
