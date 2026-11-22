<?php

declare(strict_types=1);

use Hyperf\Contract\StdoutLoggerInterface;
use Psr\Log\LogLevel;

return [
    'app_name' => 'athlete-api',
    StdoutLoggerInterface::class => [
        'log_level' => [LogLevel::ALERT, LogLevel::CRITICAL, LogLevel::EMERGENCY, LogLevel::ERROR, LogLevel::WARNING],
    ],
];
