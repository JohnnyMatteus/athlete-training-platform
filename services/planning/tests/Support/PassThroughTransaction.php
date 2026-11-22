<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\Transaction;

final class PassThroughTransaction implements Transaction
{
    public int $runs = 0;

    public function run(callable $work): mixed
    {
        ++$this->runs;

        return $work();
    }
}
