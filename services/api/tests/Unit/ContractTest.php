<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Domain\Import\AthleteId;
use App\Domain\Import\Consent;
use App\Domain\Import\FailureCode;
use App\Domain\Import\FileType;
use App\Domain\Import\ImportBatch;
use App\Domain\Import\ImportId;
use App\Domain\Import\Sha256;
use App\Domain\Shared\DomainEvent;
use App\Infrastructure\Messaging\EventEnvelope;
use DateTimeImmutable;
use Opis\JsonSchema\Validator;
use PHPUnit\Framework\TestCase;
use Tests\Support\Samples;

/** Os eventos publicados pela API precisam validar nos esquemas de contracts/events/importacao. */
final class ContractTest extends TestCase
{
    private const DIR = __DIR__ . '/../../../../contracts/events/importacao/';

    private function batch(): ImportBatch
    {
        $now = new DateTimeImmutable('2026-11-21T12:00:00Z');

        return ImportBatch::restore(
            new ImportId('3c1d2e4f-0b1a-4c2d-8e3f-9a8b7c6d5e40'),
            new AthleteId(Samples::ATHLETE),
            FileType::Zip,
            'export.zip',
            1024,
            Sha256::fromHex(Samples::SHA),
            Consent::restore(['treinos'], '2026-11', $now),
            $now,
            $now->modify('+1 day'),
            \App\Domain\Import\ImportStatus::AguardandoUpload,
            null,
            0,
            0,
            0,
            null,
            $now,
        );
    }

    private function assertValid(DomainEvent $event, string $schemaFile): void
    {
        $envelope = EventEnvelope::from($event, '0b9a8c7d-6e5f-4a3b-9c2d-1e0f9a8b7c6d');
        $schema = json_decode((string) file_get_contents(self::DIR . $schemaFile));
        $result = (new Validator())->validate(json_decode(json_encode($envelope)), $schema);

        self::assertTrue($result->isValid(), json_encode($result->error()?->message()));
    }

    public function testImportacaoRecebidaMatchesSchema(): void
    {
        $batch = $this->batch();
        $batch->markReceived(new DateTimeImmutable('2026-11-21T12:01:00Z'), 'uploads', 'tus-id');

        $this->assertValid($batch->pullEvents()[0], 'importacao-recebida.v1.schema.json');
    }

    public function testImportacaoFalhouMatchesSchema(): void
    {
        $batch = $this->batch();
        $batch->reject(FailureCode::Sha256Divergente, 'não confere', new DateTimeImmutable('2026-11-21T12:01:00Z'));

        $this->assertValid($batch->pullEvents()[0], 'importacao-falhou.v1.schema.json');
    }
}
