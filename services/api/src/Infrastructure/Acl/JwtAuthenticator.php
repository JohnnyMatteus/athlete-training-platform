<?php

declare(strict_types=1);

namespace App\Infrastructure\Acl;

use App\Application\Import\Unauthenticated;
use App\Application\Port\Authenticator;
use App\Domain\Import\AthleteId;
use Firebase\JWT\JWT;
use Firebase\JWT\Key;
use Throwable;

/**
 * ACL do provedor de identidade (ADR 0011): valida o JWT e traduz a claim "sub" em AthleteId.
 * Nenhuma classe da biblioteca de JWT atravessa a porta.
 */
final readonly class JwtAuthenticator implements Authenticator
{
    public function __construct(
        private string $publicKey,
        private string $issuer,
        private string $audience,
        private string $algorithm = 'RS256',
        private int $leewaySeconds = 30,
    ) {
    }

    public function authenticate(string $bearerToken): AthleteId
    {
        JWT::$leeway = $this->leewaySeconds;

        try {
            $claims = (array) JWT::decode($bearerToken, new Key($this->publicKey, $this->algorithm));
            $audiences = (array) ($claims['aud'] ?? []);

            if (($claims['iss'] ?? null) !== $this->issuer || !in_array($this->audience, $audiences, true)) {
                throw new Unauthenticated('emissor ou audiência inválidos');
            }
            if (!isset($claims['exp'])) {
                throw new Unauthenticated('token sem expiração');
            }

            return new AthleteId((string) ($claims['sub'] ?? ''));
        } catch (Unauthenticated $e) {
            throw $e;
        } catch (Throwable $e) {
            throw new Unauthenticated('token inválido', 0, $e);
        }
    }
}
