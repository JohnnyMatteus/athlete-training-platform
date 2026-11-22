<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\Import\AuthorizeUpload;
use App\Application\Import\ConfirmUpload;
use App\Application\Import\UploadRejected;
use App\Application\Port\Authenticator;
use App\Application\Port\ImportTraceStore;
use App\Domain\Import\ImportId;
use App\Infrastructure\Acl\TusdHook;
use App\Infrastructure\Telemetry\Tracing;
use Hyperf\Context\Context;
use Hyperf\HttpServer\Contract\RequestInterface;
use Hyperf\HttpServer\Contract\ResponseInterface;
use Psr\Http\Message\ResponseInterface as Psr7Response;
use Throwable;

/** Endpoint dos hooks HTTP do tusd (rede interna; segredo compartilhado opcional em ?token=). */
final class TusdHookController
{
    public function __construct(
        private readonly RequestInterface $request,
        private readonly ResponseInterface $response,
        private readonly AuthorizeUpload $authorizeUpload,
        private readonly ConfirmUpload $confirmUpload,
        private readonly Authenticator $authenticator,
        private readonly ImportTraceStore $traces,
        private readonly string $hookSecret,
    ) {
    }

    public function handle(): Psr7Response
    {
        if ($this->hookSecret !== '' && !hash_equals($this->hookSecret, (string) $this->request->query('token', ''))) {
            return $this->response->json(['error' => 'forbidden'])->withStatus(403);
        }

        $body = $this->request->getParsedBody();
        $hook = TusdHook::fromArray(is_array($body) ? $body : []);

        return match ($hook->type) {
            'pre-create' => $this->preCreate($hook),
            'post-finish' => $this->postFinish($hook),
            default => $this->response->json([]),
        };
    }

    private function preCreate(TusdHook $hook): Psr7Response
    {
        try {
            if ($hook->importId === null || $hook->bearerToken === null) {
                throw new UploadRejected('import_id e Authorization são obrigatórios');
            }
            ($this->authorizeUpload)(
                $this->authenticator->authenticate($hook->bearerToken),
                new ImportId($hook->importId),
                $hook->size,
            );
        } catch (Throwable $e) {
            return $this->response->json([
                'RejectUpload' => true,
                'HTTPResponse' => ['StatusCode' => 403, 'Body' => 'upload não autorizado'],
            ]);
        }

        return $this->response->json([]);
    }

    private function postFinish(TusdHook $hook): Psr7Response
    {
        if ($hook->importId === null || $hook->uploadId === null || $hook->bucket === null) {
            return $this->response->json(['error' => 'metadata incompleta'])->withStatus(422);
        }
        $id = new ImportId($hook->importId);
        // O tusd não repassa traceparent: o hook continua o trace aberto em POST /imports.
        $span = Tracing::start('tusd post-finish', $this->traces->find($id));
        Context::set(Tracing::CONTEXT_KEY, $span->traceparent());
        try {
            ($this->confirmUpload)($id, $hook->bucket, $hook->uploadId);
        } catch (Throwable $e) {
            $span->fail($e);
            throw $e;
        } finally {
            $span->end();
        }

        return $this->response->json([]);
    }
}
