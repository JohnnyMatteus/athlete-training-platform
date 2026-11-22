<?php

declare(strict_types=1);

namespace App\Domain\Import;

use App\Domain\Import\Event\ImportacaoFalhou;
use App\Domain\Import\Event\ImportacaoRecebida;
use App\Domain\Shared\DomainEvent;
use DateInterval;
use DateTimeImmutable;

/** Agregado LoteDeImportacao (contexto Importação). */
final class ImportBatch
{
    /** @var list<DomainEvent> */
    private array $events = [];

    private function __construct(
        public readonly ImportId $id,
        public readonly AthleteId $athleteId,
        public readonly FileType $type,
        public readonly string $fileName,
        public readonly int $sizeBytes,
        public readonly Sha256 $sha256,
        public readonly Consent $consent,
        public readonly DateTimeImmutable $createdAt,
        public readonly DateTimeImmutable $expiresAt,
        private ImportStatus $status,
        private ?string $failureReason,
        private int $imported,
        private int $duplicated,
        private int $errors,
        private ?string $formatVersion,
        private DateTimeImmutable $updatedAt,
    ) {
    }

    /** @param array<string, mixed> $input */
    public static function open(
        ImportId $id,
        AthleteId $athleteId,
        array $input,
        DateTimeImmutable $now,
        DateInterval $uploadTtl,
        int $maxSizeBytes,
    ): self {
        $errors = [];
        $capture = static function (callable $build, string $field) use (&$errors): mixed {
            try {
                return $build();
            } catch (ValidationFailed $e) {
                $errors += $e->errors;

                return null;
            }
        };

        $type = $capture(static fn () => FileType::fromInput($input['tipo_arquivo'] ?? null), 'tipo');
        $sha = $capture(static fn () => Sha256::fromHex($input['sha256'] ?? null), 'sha256');
        $consent = $capture(static fn () => Consent::fromInput($input['consentimento'] ?? null, $now), 'consentimento');

        $name = $input['nome_arquivo'] ?? null;
        if (!is_string($name) || trim($name) === '' || strlen($name) > 255 || preg_match('#[/\\\\\x00]#', $name) === 1) {
            $errors['nome_arquivo'] = 'nome inválido (obrigatório, até 255 caracteres, sem separadores de caminho)';
        } elseif ($type instanceof FileType && !$type->matchesFileName($name)) {
            $errors['nome_arquivo'] = sprintf('a extensão deve ser .%s', $type->value);
        }

        $size = $input['tamanho_bytes'] ?? null;
        if (!is_int($size) || $size <= 0) {
            $errors['tamanho_bytes'] = 'deve ser um inteiro positivo';
        } elseif ($size > $maxSizeBytes) {
            $errors['tamanho_bytes'] = sprintf('excede o limite de %d bytes', $maxSizeBytes);
        }

        if ($errors !== []) {
            throw new ValidationFailed($errors);
        }

        return new self(
            $id,
            $athleteId,
            $type,
            trim($name),
            $size,
            $sha,
            $consent,
            $now,
            $now->add($uploadTtl),
            ImportStatus::AguardandoUpload,
            null,
            0,
            0,
            0,
            null,
            $now,
        );
    }

    public static function restore(
        ImportId $id,
        AthleteId $athleteId,
        FileType $type,
        string $fileName,
        int $sizeBytes,
        Sha256 $sha256,
        Consent $consent,
        DateTimeImmutable $createdAt,
        DateTimeImmutable $expiresAt,
        ImportStatus $status,
        ?string $failureReason,
        int $imported,
        int $duplicated,
        int $errors,
        ?string $formatVersion,
        DateTimeImmutable $updatedAt,
    ): self {
        return new self(
            $id,
            $athleteId,
            $type,
            $fileName,
            $sizeBytes,
            $sha256,
            $consent,
            $createdAt,
            $expiresAt,
            $status,
            $failureReason,
            $imported,
            $duplicated,
            $errors,
            $formatVersion,
            $updatedAt,
        );
    }

    public function status(): ImportStatus
    {
        return $this->status;
    }

    public function failureReason(): ?string
    {
        return $this->failureReason;
    }

    public function acceptsUpload(DateTimeImmutable $now): bool
    {
        return $this->status === ImportStatus::AguardandoUpload && $now < $this->expiresAt;
    }

    public function imported(): int
    {
        return $this->imported;
    }

    public function duplicated(): int
    {
        return $this->duplicated;
    }

    public function errors(): int
    {
        return $this->errors;
    }

    public function formatVersion(): ?string
    {
        return $this->formatVersion;
    }

    public function updatedAt(): DateTimeImmutable
    {
        return $this->updatedAt;
    }

    public function isFinished(): bool
    {
        return in_array($this->status, [ImportStatus::Concluido, ImportStatus::Falhou], true);
    }

    public function markReceived(DateTimeImmutable $now, string $bucket, string $objectKey): void
    {
        $this->requireStatus([ImportStatus::AguardandoUpload], ImportStatus::Recebido);
        $this->status = ImportStatus::Recebido;
        $this->updatedAt = $now;
        $this->events[] = new ImportacaoRecebida(
            $this->id,
            $this->athleteId,
            $this->type,
            $bucket,
            $objectKey,
            $this->sizeBytes,
            $this->sha256,
            $now,
        );
    }

    /** Falha detectada pela própria API (sha256 divergente): publica ImportacaoFalhou. */
    public function reject(FailureCode $code, string $reason, DateTimeImmutable $now): void
    {
        $this->requireStatus([ImportStatus::AguardandoUpload], ImportStatus::Falhou);
        $this->status = ImportStatus::Falhou;
        $this->failureReason = $reason;
        $this->updatedAt = $now;
        $this->events[] = new ImportacaoFalhou($this->id, $this->athleteId, $code, $reason, $now);
    }

    /** Resultado do worker (ImportacaoConcluida): só atualiza o lote, sem novo evento. */
    public function complete(int $imported, int $duplicated, int $errors, ?string $formatVersion, DateTimeImmutable $now): void
    {
        $this->requireStatus([ImportStatus::Recebido, ImportStatus::Processando], ImportStatus::Concluido);
        $this->status = ImportStatus::Concluido;
        [$this->imported, $this->duplicated, $this->errors, $this->formatVersion] = [$imported, $duplicated, $errors, $formatVersion];
        $this->updatedAt = $now;
    }

    /** Resultado do worker (ImportacaoFalhou): só atualiza o lote, sem novo evento. */
    public function recordWorkerFailure(string $reason, DateTimeImmutable $now): void
    {
        $this->requireStatus([ImportStatus::Recebido, ImportStatus::Processando], ImportStatus::Falhou);
        $this->status = ImportStatus::Falhou;
        $this->failureReason = $reason;
        $this->updatedAt = $now;
    }

    /** @return list<DomainEvent> */
    public function pullEvents(): array
    {
        $events = $this->events;
        $this->events = [];

        return $events;
    }

    /** @param list<ImportStatus> $allowed */
    private function requireStatus(array $allowed, ImportStatus $to): void
    {
        if (!in_array($this->status, $allowed, true)) {
            throw InvalidTransition::from($this->status, $to);
        }
    }
}
