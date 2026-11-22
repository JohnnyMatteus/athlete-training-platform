<?php

declare(strict_types=1);

namespace App\Domain\Shared;

enum Modality: string
{
    case Swim = 'natacao';
    case Bike = 'ciclismo';
    case Run = 'corrida';
    case Strength = 'forca';

    /** Ordem de desempate determinística ao escolher a modalidade do dia. */
    public function priority(): int
    {
        return match ($this) {
            self::Swim => 0,
            self::Bike => 1,
            self::Run => 2,
            self::Strength => 3,
        };
    }
}
