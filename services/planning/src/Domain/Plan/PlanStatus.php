<?php

declare(strict_types=1);

namespace App\Domain\Plan;

enum PlanStatus: string
{
    case AwaitingApproval = 'aguardando_aprovacao';
    case Active = 'ativo';
    case Rejected = 'rejeitado';
}
