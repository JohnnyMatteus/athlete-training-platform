<?php

declare(strict_types=1);

namespace App\Infrastructure\Persistence;

use App\Application\Port\Transaction;
use Hyperf\DbConnection\Db;

final class HyperfTransaction implements Transaction
{
    public function run(callable $work): mixed
    {
        return Db::transaction($work);
    }
}
