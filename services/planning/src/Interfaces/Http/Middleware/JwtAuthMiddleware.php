<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Middleware;

use App\Application\Port\Authenticator;
use App\Application\Unauthenticated;
use Hyperf\Context\Context;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

final class JwtAuthMiddleware implements MiddlewareInterface
{
    public const ATTRIBUTE = 'principal';

    public function __construct(private readonly Authenticator $authenticator)
    {
    }

    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        if (preg_match('/^Bearer\s+(\S+)$/i', $request->getHeaderLine('Authorization'), $m) !== 1) {
            throw new Unauthenticated('cabeçalho Authorization ausente');
        }

        $request = $request->withAttribute(self::ATTRIBUTE, $this->authenticator->authenticate($m[1]));
        Context::set(ServerRequestInterface::class, $request);

        return $handler->handle($request);
    }
}
