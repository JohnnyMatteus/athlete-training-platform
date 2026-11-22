<?php

declare(strict_types=1);

namespace App\Domain\Plan;

enum Phase: string
{
    case Base = 'base';
    case Build = 'construcao';
    case Peak = 'pico';
    case Taper = 'polimento';
    case Transition = 'transicao';
}
