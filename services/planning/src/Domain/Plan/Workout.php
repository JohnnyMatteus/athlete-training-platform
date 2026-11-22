<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use App\Domain\Shared\Intensity;
use App\Domain\Shared\Modality;
use DateTimeImmutable;

final readonly class Workout
{
    /** @param list<Block> $blocks */
    public function __construct(
        public string $id,
        public DateTimeImmutable $date,
        public Modality $modality,
        public string $name,
        public Intensity $intensity,
        public int $durationS,
        public float $tss,
        public array $blocks,
    ) {
    }

    public function withDate(DateTimeImmutable $date): self
    {
        return new self($this->id, $date, $this->modality, $this->name, $this->intensity, $this->durationS, $this->tss, $this->blocks);
    }

    public function day(): string
    {
        return $this->date->format('Y-m-d');
    }

    /** @return array<string, mixed> */
    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'data' => $this->day(),
            'modalidade' => $this->modality->value,
            'nome' => $this->name,
            'intensidade' => $this->intensity->value,
            'duracao_s' => $this->durationS,
            'tss_planejado' => $this->tss,
            'blocos' => array_map(static fn (Block $b): array => $b->toArray(), $this->blocks),
        ];
    }

    /** @param array<string, mixed> $a */
    public static function fromArray(array $a): self
    {
        return new self(
            $a['id'],
            new DateTimeImmutable($a['data'] . 'T00:00:00Z'),
            Modality::from($a['modalidade']),
            $a['nome'],
            Intensity::from($a['intensidade']),
            (int) $a['duracao_s'],
            (float) $a['tss_planejado'],
            array_map(Block::fromArray(...), $a['blocos']),
        );
    }
}
