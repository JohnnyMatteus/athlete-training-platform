<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\EventOutbox;
use App\Application\Port\ImportBatchRepository;
use App\Application\Port\ObjectChecksum;
use App\Application\Port\Transaction;
use App\Domain\Import\ImportBatch;
use App\Domain\Import\ImportId;
use App\Domain\Shared\DomainEvent;
use DateTimeImmutable;
use Psr\Clock\ClockInterface;

final class Samples
{
    public const ATHLETE = '0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b';
    public const SHA = 'a3f1c2d4e5b60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90';

    /** @return array<string, mixed> */
    public static function input(array $override = []): array
    {
        return $override + [
            'tipo' => 'zip',
            'nome_arquivo' => 'export.zip',
            'tamanho_bytes' => 200_000_000,
            'sha256' => self::SHA,
            'consentimento' => [
                'aceito' => true,
                'versao_termo' => '2026-11',
                'categorias' => ['treinos', 'frequencia_cardiaca'],
            ],
        ];
    }
}
