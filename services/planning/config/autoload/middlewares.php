<?php

declare(strict_types=1);

use App\Interfaces\Http\Middleware\TraceMiddleware;

return ['http' => [TraceMiddleware::class]];
