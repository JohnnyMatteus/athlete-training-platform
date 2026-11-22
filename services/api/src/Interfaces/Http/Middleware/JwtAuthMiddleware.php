<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Middleware;

use App\Application\Import\Unauthenticated;
use App\Application\Port\Authenticator;
use Hyperf\Context\Context;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

final class JwtAuthMiddleware implements MiddlewareInterface
{
    public const ATTRIBUTE = 'athlete';

    public function __construct(private readonly Authenticator $authenticator)
    {
    }

    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        $header = $request->getHeaderLine('Authorization');
        if (preg_match('/^Bearer\s+(\S+)$/i', $header, $m) !== 1) {
            throw new Unauthenticated('cabeçalho Authorization ausente');
        }

        $request = $request->withAttribute(self::ATTRIBUTE, $this->authenticator->authenticate($m[1]));
        Context::set(ServerRequestInterface::class, $request);

        return $handler->handle($request);
    }
}
