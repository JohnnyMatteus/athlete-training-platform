<?php

declare(strict_types=1);

namespace App\Infrastructure\Persistence;

use App\Application\Port\Inbox;
use Hyperf\DbConnection\Db;

final class PgInbox implements Inbox
{
    public function firstTime(string $eventId): bool
    {
        return Db::affectingStatement('INSERT INTO evento_processado (event_id) VALUES (?) ON CONFLICT DO NOTHING', [$eventId]) === 1;
    }
}
