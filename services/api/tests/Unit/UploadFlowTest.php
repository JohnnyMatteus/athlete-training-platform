<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Application\Import\AuthorizeUpload;
use App\Application\Import\ConfirmUpload;
use App\Application\Import\CreateImport;
use App\Application\Import\UploadRejected;
use App\Domain\Import\AthleteId;
use App\Domain\Import\Event\ImportacaoFalhou;
use App\Domain\Import\Event\ImportacaoRecebida;
use App\Domain\Import\ImportId;
use App\Domain\Import\ImportStatus;
use DateInterval;
use PHPUnit\Framework\TestCase;
use Tests\Support\FakeChecksum;
use Tests\Support\FixedClock;
use Tests\Support\InMemoryBatches;
use Tests\Support\InMemoryOutbox;
use Tests\Support\PassThroughTransaction;
use Tests\Support\Samples;

final class UploadFlowTest extends TestCase
{
    private InMemoryBatches $batches;
    private InMemoryOutbox $outbox;
    private FixedClock $clock;
    private ImportId $id;
    private AthleteId $athlete;

    protected function setUp(): void
    {
        $this->batches = new InMemoryBatches();
        $this->outbox = new InMemoryOutbox();
        $this->clock = new FixedClock();
        $this->athlete = new AthleteId(Samples::ATHLETE);

        $result = (new CreateImport($this->batches, $this->clock, 'http://tus/', 2 * 1024 ** 3, new DateInterval('PT24H')))(
            $this->athlete,
            Samples::input(),
        );
        $this->id = new ImportId($result['import_id']);
    }

    private function confirm(string $actualSha): void
    {
        (new ConfirmUpload($this->batches, new FakeChecksum($actualSha), $this->outbox, new PassThroughTransaction(), $this->clock))(
            $this->id,
            'objeto-tus',
        );
    }

    public function testMatchingChecksumMarksReceivedAndPublishesEvent(): void
    {
        $this->confirm(Samples::SHA);

        self::assertSame(ImportStatus::Recebido, $this->batches->find($this->id)->status());
        self::assertCount(1, $this->outbox->events);
        self::assertInstanceOf(ImportacaoRecebida::class, $this->outbox->events[0]);
        self::assertSame(['import_id' => $this->id->value], $this->outbox->events[0]->payload());
    }

    public function testChecksumMismatchFailsBatchAndPublishesOnlyFailure(): void
    {
        $this->confirm(str_repeat('0', 64));

        $batch = $this->batches->find($this->id);
        self::assertSame(ImportStatus::Falhou, $batch->status());
        self::assertNotEmpty($batch->failureReason());
        self::assertCount(1, $this->outbox->events);
        self::assertInstanceOf(ImportacaoFalhou::class, $this->outbox->events[0]);
    }

    public function testRepeatedHookIsIdempotent(): void
    {
        $this->confirm(Samples::SHA);
        $this->confirm(Samples::SHA);

        self::assertCount(1, $this->outbox->events);
    }

    public function testUploadAuthorizedForOwnerOpenBatchAndDeclaredSize(): void
    {
        $authorize = new AuthorizeUpload($this->batches, $this->clock);
        $authorize($this->athlete, $this->id, 200_000_000);

        $this->addToAssertionCount(1);
    }

    public function testUploadRejectedForOtherAthlete(): void
    {
        $this->expectException(UploadRejected::class);
        (new AuthorizeUpload($this->batches, $this->clock))(
            new AthleteId('0190a1b2-c3d4-7e5f-8a9b-ffffffffffff'),
            $this->id,
            200_000_000,
        );
    }

    public function testUploadRejectedAfterCompletion(): void
    {
        $this->confirm(Samples::SHA);

        $this->expectException(UploadRejected::class);
        (new AuthorizeUpload($this->batches, $this->clock))($this->athlete, $this->id, 200_000_000);
    }

    public function testUploadRejectedAfterExpiry(): void
    {
        $this->clock->now = $this->clock->now->modify('+25 hours');

        $this->expectException(UploadRejected::class);
        (new AuthorizeUpload($this->batches, $this->clock))($this->athlete, $this->id, 200_000_000);
    }

    public function testUploadRejectedWhenSizeDiffersFromDeclared(): void
    {
        $this->expectException(UploadRejected::class);
        (new AuthorizeUpload($this->batches, $this->clock))($this->athlete, $this->id, 1);
    }
}
