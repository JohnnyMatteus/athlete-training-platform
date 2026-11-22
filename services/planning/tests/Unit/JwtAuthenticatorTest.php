<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Application\Unauthenticated;
use App\Infrastructure\Acl\JwtAuthenticator;
use Firebase\JWT\JWT;
use PHPUnit\Framework\TestCase;

final class JwtAuthenticatorTest extends TestCase
{
    private const SUB = '2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01';
    private string $private = '';
    private JwtAuthenticator $auth;

    protected function setUp(): void
    {
        $key = openssl_pkey_new(['private_key_bits' => 2048, 'private_key_type' => OPENSSL_KEYTYPE_RSA]);
        openssl_pkey_export($key, $this->private);
        $public = openssl_pkey_get_details($key)['key'];
        $this->auth = new JwtAuthenticator($public, 'https://idp.example.test/', 'athlete-api');
    }

    /** @param array<string, mixed> $claims */
    private function token(array $claims = []): string
    {
        return JWT::encode($claims + ['iss' => 'https://idp.example.test/', 'aud' => 'athlete-api', 'sub' => self::SUB, 'exp' => time() + 60], $this->private, 'RS256');
    }

    public function testRolesDefaultToAthleteAndUnknownRolesAreDropped(): void
    {
        self::assertSame(['atleta'], $this->auth->authenticate($this->token())->roles);
        $p = $this->auth->authenticate($this->token(['roles' => ['treinador', 'superuser', 'atleta']]));

        self::assertSame(['treinador', 'atleta'], $p->roles);
        self::assertTrue($p->isCoach());
        self::assertSame(self::SUB, $p->id);
    }

    public function testRejectsBadIssuerAudienceSubjectAndMissingExpiry(): void
    {
        foreach ([['iss' => 'outro'], ['aud' => 'x'], ['sub' => 'não-uuid']] as $bad) {
            try {
                $this->auth->authenticate($this->token($bad));
                self::fail('deveria rejeitar ' . json_encode($bad));
            } catch (Unauthenticated) {
            }
        }
        $noExp = JWT::encode(['iss' => 'https://idp.example.test/', 'aud' => 'athlete-api', 'sub' => self::SUB], $this->private, 'RS256');
        $this->expectException(Unauthenticated::class);
        $this->auth->authenticate($noExp);
    }
}
