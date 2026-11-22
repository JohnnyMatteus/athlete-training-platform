<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Application\Import\Unauthenticated;
use App\Infrastructure\Acl\JwtAuthenticator;
use Firebase\JWT\JWT;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;
use Tests\Support\Samples;

final class JwtAuthenticatorTest extends TestCase
{
    private string $privateKey;
    private JwtAuthenticator $auth;

    protected function setUp(): void
    {
        $key = openssl_pkey_new(['private_key_bits' => 2048, 'private_key_type' => OPENSSL_KEYTYPE_RSA]);
        openssl_pkey_export($key, $private);
        $this->privateKey = $private;
        $public = openssl_pkey_get_details($key)['key'];
        $this->auth = new JwtAuthenticator($public, 'https://idp.test/', 'athlete-api');
    }

    /** @param array<string, mixed> $claims */
    private function token(array $claims, ?string $privateKey = null): string
    {
        return JWT::encode($claims + [
            'iss' => 'https://idp.test/',
            'aud' => 'athlete-api',
            'sub' => Samples::ATHLETE,
            'exp' => time() + 300,
        ], $privateKey ?? $this->privateKey, 'RS256');
    }

    public function testValidTokenYieldsAthleteId(): void
    {
        self::assertSame(Samples::ATHLETE, $this->auth->authenticate($this->token([]))->value);
    }

    /** @return iterable<string, array{array<string, mixed>}> */
    public static function badClaims(): iterable
    {
        yield 'expirado' => [['exp' => time() - 3600]];
        yield 'emissor errado' => [['iss' => 'https://evil.test/']];
        yield 'audiência errada' => [['aud' => 'outra-api']];
        yield 'sub não é uuid' => [['sub' => 'admin']];
    }

    /** @param array<string, mixed> $claims */
    #[DataProvider('badClaims')]
    public function testRejectsInvalidClaims(array $claims): void
    {
        $this->expectException(Unauthenticated::class);
        $this->auth->authenticate($this->token($claims));
    }

    public function testRejectsTokenSignedWithAnotherKey(): void
    {
        $other = openssl_pkey_new(['private_key_bits' => 2048, 'private_key_type' => OPENSSL_KEYTYPE_RSA]);
        openssl_pkey_export($other, $otherPrivate);

        $this->expectException(Unauthenticated::class);
        $this->auth->authenticate($this->token([], $otherPrivate));
    }

    public function testRejectsGarbage(): void
    {
        $this->expectException(Unauthenticated::class);
        $this->auth->authenticate('nao-e-um-jwt');
    }
}
