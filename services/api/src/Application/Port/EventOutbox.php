<?php

declare(strict_types=1);

namespace App\Application\Port;

use App\Domain\Shared\DomainEvent;

/** Grava o evento na mesma transação do agregado; um relay o publica no broker depois do commit. */
interface EventOutbox
{
    public function append(DomainEvent $event): void;
}
