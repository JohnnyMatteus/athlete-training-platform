<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\Conflict;
use App\Application\Forbidden;
use App\Application\NotFound;
use App\Application\PainBlocked;
use App\Application\Unauthenticated;
use App\Application\ValidationFailed;
use App\Domain\Plan\ConstraintsNotSatisfiable;
use App\Domain\Plan\Violation;
use Hyperf\ExceptionHandler\ExceptionHandler;
use Hyperf\HttpMessage\Stream\SwooleStream;
use Psr\Http\Message\ResponseInterface;
use Throwable;

/** Erros no formato RFC 9457 (application/problem+json), como no OpenAPI. */
final class ProblemExceptionHandler extends ExceptionHandler
{
    public function handle(Throwable $throwable, ResponseInterface $response): ResponseInterface
    {
        [$status, $title, $extra] = match (true) {
            $throwable instanceof ValidationFailed => [422, 'Dados inválidos', ['errors' => array_map(
                static fn (string $field, string $message): array => ['campo' => $field, 'mensagem' => $message],
                array_keys($throwable->errors),
                $throwable->errors,
            )]],
            $throwable instanceof PainBlocked => [422, 'Geração bloqueada', [
                'codigo' => 'bloqueado_por_dor',
                'detail' => 'Dor ou lesão relatada. O sistema não prescreve nesse caso.',
                'recomendacao' => 'Procure um médico ou fisioterapeuta antes de retomar o treino.',
                'violacoes' => [$throwable->violation()->toArray()],
            ]],
            $throwable instanceof ConstraintsNotSatisfiable => [422, 'Restrições violadas', [
                'codigo' => 'restricoes_violadas',
                'violacoes' => array_map(static fn (Violation $v): array => $v->toArray(), $throwable->violations),
            ]],
            $throwable instanceof Unauthenticated => [401, 'Não autenticado', []],
            $throwable instanceof Forbidden => [403, 'Sem permissão', []],
            $throwable instanceof NotFound => [404, 'Não encontrado', []],
            $throwable instanceof Conflict => [409, 'Conflito', ['detail' => $throwable->getMessage()]],
            default => [500, 'Erro interno', []],
        };

        $this->stopPropagation();

        return $response
            ->withStatus($status)
            ->withHeader('Content-Type', 'application/problem+json')
            ->withBody(new SwooleStream(json_encode(['type' => 'about:blank', 'title' => $title, 'status' => $status] + $extra, JSON_THROW_ON_ERROR)));
    }

    public function isValid(Throwable $throwable): bool
    {
        return true;
    }
}
