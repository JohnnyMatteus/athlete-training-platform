<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\Import\CreateImport;
use App\Application\Import\GetImport;
use App\Domain\Import\AthleteId;
use App\Domain\Import\ImportId;
use App\Domain\Import\ValidationFailed;
use App\Interfaces\Http\Middleware\JwtAuthMiddleware;
use Hyperf\HttpServer\Contract\RequestInterface;
use Hyperf\HttpServer\Contract\ResponseInterface;
use Psr\Http\Message\ResponseInterface as Psr7Response;

final class ImportController
{
    public function __construct(
        private readonly RequestInterface $request,
        private readonly ResponseInterface $response,
        private readonly CreateImport $createImport,
        private readonly GetImport $getImport,
    ) {
    }

    public function create(): Psr7Response
    {
        $body = $this->request->getParsedBody();
        $result = ($this->createImport)($this->athlete(), is_array($body) ? $body : []);

        return $this->response->json($result)->withStatus(201);
    }

    public function show(string $id): Psr7Response
    {
        try {
            $importId = new ImportId($id);
        } catch (\InvalidArgumentException) {
            throw ValidationFailed::field('id', 'deve ser um UUID');
        }

        return $this->response->json(($this->getImport)($this->athlete(), $importId));
    }

    private function athlete(): AthleteId
    {
        return $this->request->getAttribute(JwtAuthMiddleware::ATTRIBUTE);
    }
}
