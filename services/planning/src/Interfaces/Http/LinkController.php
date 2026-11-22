<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\Link\LinkService;
use App\Application\NotFound;
use App\Domain\Link\Link;
use App\Domain\Shared\Principal;
use App\Interfaces\Http\Middleware\JwtAuthMiddleware;
use Hyperf\HttpServer\Contract\RequestInterface;
use Hyperf\HttpServer\Contract\ResponseInterface;
use Psr\Http\Message\ResponseInterface as Psr7Response;
use Ramsey\Uuid\Uuid;

final class LinkController
{
    public function __construct(
        private readonly RequestInterface $request,
        private readonly ResponseInterface $response,
        private readonly LinkService $links,
    ) {
    }

    public function invite(): Psr7Response
    {
        $body = $this->request->getParsedBody();
        $body = is_array($body) ? $body : [];
        $link = $this->links->invite($this->who(), strtolower((string) ($body['treinador_id'] ?? '')), is_array($body['categorias'] ?? null) ? $body['categorias'] : []);

        return $this->response->json($link->toArray())->withStatus(201);
    }

    public function index(): Psr7Response
    {
        return $this->response->json(['itens' => array_map(static fn (Link $l): array => $l->toArray(), $this->links->list($this->who()))]);
    }

    public function accept(string $id): Psr7Response
    {
        return $this->response->json($this->links->accept($this->who(), $this->uuid($id))->toArray());
    }

    public function revoke(string $id): Psr7Response
    {
        $this->links->revoke($this->who(), $this->uuid($id));

        return $this->response->raw('')->withStatus(204);
    }

    private function who(): Principal
    {
        return $this->request->getAttribute(JwtAuthMiddleware::ATTRIBUTE);
    }

    private function uuid(string $id): string
    {
        if (!Uuid::isValid($id)) {
            throw new NotFound('vínculo não encontrado');
        }

        return strtolower($id);
    }
}
