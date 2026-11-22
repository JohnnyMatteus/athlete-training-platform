<?php

declare(strict_types=1);

namespace App\Application\Import;

use App\Application\Port\ImportBatchRepository;
use App\Domain\Import\AthleteId;
use App\Domain\Import\ImportBatch;
use App\Domain\Import\ImportId;
use DateInterval;
use Psr\Clock\ClockInterface;
use Ramsey\Uuid\Uuid;

final readonly class CreateImport
{
    public function __construct(
        private ImportBatchRepository $batches,
        private ClockInterface $clock,
        private string $tusEndpoint,
        private int $maxSizeBytes,
        private DateInterval $uploadTtl,
    ) {
    }

    /**
     * @param array<string, mixed> $input corpo do POST /imports
     * @return array{import_id: string, upload_url: string, upload_metadata: array<string, string>, expira_em: string}
     */
    public function __invoke(AthleteId $athlete, array $input): array
    {
        $batch = ImportBatch::open(
            new ImportId(Uuid::uuid7()->toString()),
            $athlete,
            $input,
            $this->clock->now(),
            $this->uploadTtl,
            $this->maxSizeBytes,
        );
        $this->batches->save($batch);

        return [
            'import_id' => $batch->id->value,
            'upload_url' => $this->tusEndpoint,
            'upload_metadata' => ['import_id' => $batch->id->value],
            'expira_em' => $batch->expiresAt->format(DATE_ATOM),
        ];
    }
}
