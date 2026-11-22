<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Interfaces\Http\HookSecret;
use PHPUnit\Framework\TestCase;

final class HookSecretTest extends TestCase
{
    public function testAcceptsTheConfiguredSecret(): void
    {
        self::assertTrue((new HookSecret('dev-only-hook'))->accepts('dev-only-hook'));
    }

    public function testRejectsAWrongToken(): void
    {
        self::assertFalse((new HookSecret('dev-only-hook'))->accepts('outro'));
        self::assertFalse((new HookSecret('dev-only-hook'))->accepts(''));
    }

    public function testEmptySecretFailsClosedEvenForAnEmptyToken(): void
    {
        self::assertFalse((new HookSecret(''))->accepts(''));
        self::assertFalse((new HookSecret(''))->accepts('qualquer'));
    }

    public function testSecretDoesNotLeakThroughVarDump(): void
    {
        ob_start();
        var_dump(new HookSecret('dev-only-hook'));

        self::assertStringNotContainsString('dev-only-hook', (string) ob_get_clean());
    }
}
