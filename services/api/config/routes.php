<?php

declare(strict_types=1);

use App\Interfaces\Http\ImportController;
use App\Interfaces\Http\Middleware\JwtAuthMiddleware;
use App\Interfaces\Http\TusdHookController;
use Hyperf\HttpServer\Router\Router;

Router::addGroup('/imports', static function (): void {
    Router::post('', [ImportController::class, 'create']);
    Router::get('/{id}', [ImportController::class, 'show']);
}, ['middleware' => [JwtAuthMiddleware::class]]);

Router::post('/hooks/tus', [TusdHookController::class, 'handle']);
Router::get('/health', static fn () => ['status' => 'ok']);
