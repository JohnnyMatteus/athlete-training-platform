<?php

declare(strict_types=1);

use App\Interfaces\Http\LinkController;
use App\Interfaces\Http\Middleware\JwtAuthMiddleware;
use App\Interfaces\Http\PlanController;
use Hyperf\HttpServer\Router\Router;

Router::addGroup('', static function (): void {
    Router::post('/planos', [PlanController::class, 'create']);
    Router::get('/planos', [PlanController::class, 'index']);
    Router::get('/planos/{id}', [PlanController::class, 'show']);
    Router::post('/planos/{id}/aprovacao', [PlanController::class, 'decision']);
    Router::post('/planos/{id}/ajustes', [PlanController::class, 'adjust']);
    Router::post('/planos/{id}/ajustes/validacao', [PlanController::class, 'validateAdjust']);
    Router::get('/planos/{id}/aderencia', [PlanController::class, 'adherence']);
    Router::get('/treinos/{id}/export.fit', [PlanController::class, 'exportFit']);

    Router::post('/vinculos', [LinkController::class, 'invite']);
    Router::get('/vinculos', [LinkController::class, 'index']);
    Router::post('/vinculos/{id}/aceite', [LinkController::class, 'accept']);
    Router::delete('/vinculos/{id}', [LinkController::class, 'revoke']);
}, ['middleware' => [JwtAuthMiddleware::class]]);

Router::get('/health', static fn () => ['status' => 'ok']);
