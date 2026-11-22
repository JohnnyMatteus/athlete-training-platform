<?php

declare(strict_types=1);

return [
    'default' => [
        'driver' => 'pgsql',
        'host' => env('DB_HOST', 'postgres'),
        'port' => (int) env('DB_PORT', 5432),
        'database' => env('DB_DATABASE', 'athlete'),
        'username' => env('DB_USERNAME', 'athlete'),
        'password' => env('DB_PASSWORD', ''),
        'charset' => 'utf8',
        'pool' => ['min_connections' => 1, 'max_connections' => 10, 'connect_timeout' => 10.0, 'wait_timeout' => 3.0],
    ],
];
