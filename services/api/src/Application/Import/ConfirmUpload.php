<?php

declare(strict_types=1);

namespace App\Application\Import;

use App\Application\Port\EventOutbox;
use App\Application\Port\ImportBatchRepository;
use App\Application\Port\ObjectChecksum;
use App\Application\Port\Transaction;
use App\Domain\Import\FailureCode;
use App\Domain\Import\ImportId;
use App\Domain\Import\ImportStatus;
use App\Domain\Import\Sha256;
use Psr\Clock\ClockInterface;

/** Hook post-finish do tusd: confere o sha256 e publica ImportacaoRecebida ou ImportacaoFalhou pela outbox. */
final readonly class ConfirmUpload
{
    public function __construct(
        private ImportBatchRepository $batches,
        private ObjectChecksum $checksum,
        private EventOutbox $outbox,
        private Transaction $transaction,
        private ClockInterface $clock,
    ) {
    }

    /** @param string $uploadId ID do upload no tusd; é também a chave do objeto no bucket (object_key do evento) */
    public function __invoke(ImportId $id, string $bucket, string $uploadId): void
    {
        $batch = $this->batches->find($id) ?? throw new NotFound('Importação não encontrada');
        if ($batch->status() !== ImportStatus::AguardandoUpload) {
            return; // hook reenviado pelo tusd: já tratado
        }
        $now = $this->clock->now();

        // Calculado fora da transação: ler o objeto pode levar minutos.
        $actual = Sha256::fromHex($this->checksum->sha256($bucket, $uploadId));

        if ($actual->equals($batch->sha256)) {
            $batch->markReceived($now, $bucket, $uploadId);
        } else {
            $batch->reject(
                FailureCode::Sha256Divergente,
                'O conteúdo enviado não confere com o sha256 declarado.',
                $now,
            );
        }

        $this->transaction->run(function () use ($batch): void {
            $this->batches->save($batch);
            foreach ($batch->pullEvents() as $event) {
                $this->outbox->append($event);
            }
        });
    }
}
