<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\Import\NotFound;
use App\Application\Import\Unauthenticated;
use App\Domain\Import\ValidationFailed;
use Hyperf\ExceptionHandler\ExceptionHandler;
use Hyperf\HttpMessage\Stream\SwooleStream;
use Psr\Http\Message\ResponseInterface;
use Throwable;

/** Erros no formato RFC 7807 (application/problem+json), como descrito no OpenAPI. */
final class ProblemExceptionHandler extends ExceptionHandler
{
    public function handle(Throwable $throwable, ResponseInterface $response): ResponseInterface
    {
        [$status, $title, $extra] = match (true) {
            $throwable instanceof ValidationFailed => [422, 'Dados inválidos', ['errors' => $throwable->errors]],
            $throwable instanceof Unauthenticated => [401, 'Não autenticado', []],
            $throwable instanceof NotFound => [404, 'Não encontrado', []],
            default => [500, 'Erro interno', []],
        };

        $this->stopPropagation();

        return $response
            ->withStatus($status)
            ->withHeader('Content-Type', 'application/problem+json')
            ->withBody(new SwooleStream(json_encode(['title' => $title, 'status' => $status] + $extra, JSON_THROW_ON_ERROR)));
    }

    public function isValid(Throwable $throwable): bool
    {
        return true;
    }
}
