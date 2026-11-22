<?php

declare(strict_types=1);

namespace App\Domain\Plan;

final readonly class Violation
{
    public const RAMP = 'rampa_ctl';
    public const TSB = 'tsb_minimo';
    public const CONSECUTIVE_HARD = 'dias_alta_consecutivos';
    public const NO_REST = 'sem_descanso';
    public const VOLUME = 'volume_acima_disponibilidade';
    public const STRENGTH_RUN = 'forca_com_limiar_corrida';
    public const PAIN = 'dor_bloqueia';

    public function __construct(
        public string $rule,
        public bool $mandatory,
        public string $message,
        public ?int $week = null,
        public ?string $date = null,
        public ?float $value = null,
        public ?float $limit = null,
    ) {
    }

    /** @return array<string, mixed> */
    public function toArray(): array
    {
        return array_filter([
            'regra' => $this->rule,
            'severidade' => $this->mandatory ? 'obrigatoria' : 'preferencial',
            'semana' => $this->week,
            'data' => $this->date,
            'valor' => $this->value,
            'limite' => $this->limit,
            'mensagem' => $this->message,
        ], static fn (mixed $v): bool => $v !== null);
    }
}
