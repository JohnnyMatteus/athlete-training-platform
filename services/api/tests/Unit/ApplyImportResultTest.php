<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Application\Import\ApplyImportResult;
use App\Application\Import\CreateImport;
use App\Application\Import\NotFound;
use App\Domain\Import\AthleteId;
use App\Domain\Import\ImportId;
use App\Domain\Import\ImportStatus;
use App\Infrastructure\Acl\ImportResultMessage;
use DateInterval;
use InvalidArgumentException;
use PHPUnit\Framework\TestCase;
use Tests\Support\FixedClock;
use Tests\Support\InMemoryBatches;
use Tests\Support\Samples;

final class ApplyImportResultTest extends TestCase
{
    private InMemoryBatches $batches;
    private ApplyImportResult $apply;
    private ImportId $id;

    protected function setUp(): void
    {
        $this->batches = new InMemoryBatches();
        $clock = new FixedClock();
        $created = (new CreateImport($this->batches, $clock, 'http://tus/', 2 * 1024 ** 3, new DateInterval('PT24H')))(
            new AthleteId(Samples::ATHLETE),
            Samples::input(),
        );
        $this->id = new ImportId($created['import_id']);
        $this->batches->find($this->id)->markReceived($clock->now(), 'uploads', 'u1');
        $this->apply = new ApplyImportResult($this->batches, $clock);
    }

    public function testCompletedStoresTotals(): void
    {
        $this->apply->completed($this->id, 3, 1, 0, 'apple-health-export-2024');

        $batch = $this->batches->find($this->id);
        self::assertSame(ImportStatus::Concluido, $batch->status());
        self::assertSame([3, 1, 0], [$batch->imported(), $batch->duplicated(), $batch->errors()]);
        self::assertSame('apple-health-export-2024', $batch->formatVersion());
    }

    public function testRedeliveryAfterFinishIsIgnored(): void
    {
        $this->apply->completed($this->id, 3, 1, 0, null);
        $this->apply->completed($this->id, 9, 9, 9, null);
        $this->apply->failed($this->id, 'tarde demais');

        $batch = $this->batches->find($this->id);
        self::assertSame(ImportStatus::Concluido, $batch->status());
        self::assertSame(3, $batch->imported());
    }

    public function testWorkerFailureRecordsReason(): void
    {
        $this->apply->failed($this->id, 'arquivo corrompido');

        self::assertSame(ImportStatus::Falhou, $this->batches->find($this->id)->status());
        self::assertSame('arquivo corrompido', $this->batches->find($this->id)->failureReason());
    }

    public function testUnknownBatchIsNotFound(): void
    {
        $this->expectException(NotFound::class);
        $this->apply->failed(new ImportId('0190a1b2-c3d4-7e5f-8a9b-ffffffffffff'), 'x');
    }

    public function testMessageTranslation(): void
    {
        $m = ImportResultMessage::fromJson(json_encode([
            'tipo' => 'ImportacaoConcluida', 'import_id' => $this->id->value, 'dados' => ['importadas' => 1],
        ]));
        self::assertSame('ImportacaoConcluida', $m->type);

        $this->expectException(InvalidArgumentException::class);
        ImportResultMessage::fromJson('{"tipo":"Outro"}');
    }
}
