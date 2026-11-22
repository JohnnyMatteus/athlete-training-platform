<?php

declare(strict_types=1);

// Registro explícito: sem annotations.php o Hyperf não varre src/ e não enxerga #[Command].
return [
    App\Interfaces\Console\ImportResultConsumerCommand::class,
    App\Interfaces\Console\OutboxRelayCommand::class,
    App\Interfaces\Console\WorkerHealthCommand::class,
];
