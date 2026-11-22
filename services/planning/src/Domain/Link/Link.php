<?php

declare(strict_types=1);

namespace App\Domain\Link;

use App\Domain\Shared\Event;
use DateTimeImmutable;
use DomainException;

/** Vínculo atleta-treinador: o atleta convida (consente), o treinador aceita, qualquer um revoga (ADR 0014). */
final readonly class Link
{
    public const CATEGORIES = ['treinos', 'frequencia_cardiaca', 'plano'];

    /** @param list<string> $categories */
    public function __construct(
        public string $id,
        public string $athleteId,
        public string $coachId,
        public string $status,
        public array $categories,
    ) {
    }

    /** @param list<string> $categories */
    public static function invite(string $id, string $athleteId, string $coachId, array $categories): self
    {
        return new self($id, $athleteId, $coachId, 'pendente', $categories);
    }

    public function accept(): self
    {
        if ($this->status !== 'pendente') {
            throw new DomainException('convite não está pendente');
        }

        return new self($this->id, $this->athleteId, $this->coachId, 'ativo', $this->categories);
    }

    public function revoke(): self
    {
        return new self($this->id, $this->athleteId, $this->coachId, 'revogado', $this->categories);
    }

    public function allows(string $category): bool
    {
        return $this->status === 'ativo' && in_array($category, $this->categories, true);
    }

    public function consentedEvent(DateTimeImmutable $at): Event
    {
        return new Event('VinculoTreinadorConsentido', 'atleta', 'vinculo.consentido', $this->id, $at, [
            'vinculo_id' => $this->id, 'atleta_id' => $this->athleteId, 'treinador_id' => $this->coachId, 'categorias' => $this->categories,
        ]);
    }

    public function revokedEvent(string $by, DateTimeImmutable $at): Event
    {
        return new Event('VinculoTreinadorRevogado', 'atleta', 'vinculo.revogado', $this->id, $at, [
            'vinculo_id' => $this->id, 'atleta_id' => $this->athleteId, 'treinador_id' => $this->coachId, 'revogado_por' => $by,
        ]);
    }

    /** @return array<string, mixed> */
    public function toArray(): array
    {
        return ['id' => $this->id, 'atleta_id' => $this->athleteId, 'treinador_id' => $this->coachId, 'status' => $this->status, 'categorias' => $this->categories];
    }
}
