<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Middleware;

use App\Infrastructure\Telemetry\Tracing;
use Hyperf\Context\Context;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;
use Throwable;

/** Abre o span do servidor com o traceparent recebido e o deixa no contexto da corrotina para a outbox. */
final class TraceMiddleware implements MiddlewareInterface
{
    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        $path = $request->getUri()->getPath();
        $span = Tracing::start($request->getMethod() . ' ' . $path, $request->getHeaderLine('traceparent') ?: null);
        Context::set(Tracing::CONTEXT_KEY, $span->traceparent());
        $span->attribute('http.request.method', $request->getMethod());
        $span->attribute('url.path', $path);

        try {
            $response = $handler->handle($request);
            $span->attribute('http.response.status_code', $response->getStatusCode());

            return $response;
        } catch (Throwable $e) {
            $span->fail($e);
            throw $e;
        } finally {
            $span->end();
        }
    }
}
