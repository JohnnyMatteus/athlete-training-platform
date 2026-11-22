<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\Plan\ApplyAdjustment;
use App\Application\Plan\CalculateAdherence;
use App\Application\Plan\DecidePlan;
use App\Application\Plan\ExportWorkoutFit;
use App\Application\Plan\GeneratePlan;
use App\Application\Plan\GetPlan;
use App\Application\Plan\ValidateAdjustment;
use App\Application\ValidationFailed;
use App\Domain\Plan\Violation;
use App\Domain\Shared\Principal;
use App\Interfaces\Http\Middleware\JwtAuthMiddleware;
use Hyperf\HttpServer\Contract\RequestInterface;
use Hyperf\HttpServer\Contract\ResponseInterface;
use Psr\Http\Message\ResponseInterface as Psr7Response;
use Ramsey\Uuid\Uuid;

final class PlanController
{
    public function __construct(
        private readonly RequestInterface $request,
        private readonly ResponseInterface $response,
        private readonly GeneratePlan $generate,
        private readonly GetPlan $get,
        private readonly DecidePlan $decide,
        private readonly ApplyAdjustment $apply,
        private readonly ValidateAdjustment $validate,
        private readonly CalculateAdherence $adherence,
        private readonly ExportWorkoutFit $export,
    ) {
    }

    public function create(): Psr7Response
    {
        $plan = ($this->generate)($this->who(), $this->body());

        return $this->response->json(PlanPresenter::full($plan))->withStatus(201)->withHeader('Location', '/planos/' . $plan->id);
    }

    public function index(): Psr7Response
    {
        $athlete = $this->request->query('atleta_id');
        if ($athlete !== null && !Uuid::isValid((string) $athlete)) {
            throw new ValidationFailed(['atleta_id' => 'Deve ser um UUID.']);
        }

        return $this->response->json(['itens' => array_map(PlanPresenter::summary(...), $this->get->list($this->who(), $athlete === null ? null : strtolower((string) $athlete)))]);
    }

    public function show(string $id): Psr7Response
    {
        return $this->response->json(PlanPresenter::full(($this->get)($this->who(), $this->uuid($id))));
    }

    public function decision(string $id): Psr7Response
    {
        $plan = ($this->decide)($this->who(), $this->uuid($id), (string) ($this->body()['decisao'] ?? ''));

        return $this->response->json(PlanPresenter::full($plan));
    }

    public function adjust(string $id): Psr7Response
    {
        $r = ($this->apply)($this->who(), $this->uuid($id), $this->body());
        $out = ['aceito' => $r['aceito'], 'violacoes' => array_map(static fn (Violation $v): array => $v->toArray(), $r['violacoes'])];
        if ($r['plano'] !== null) {
            $out['plano'] = PlanPresenter::full($r['plano']);
        }

        return $this->response->json($out);
    }

    public function validateAdjust(string $id): Psr7Response
    {
        return $this->response->json(($this->validate)($this->who(), $this->uuid($id), $this->body()));
    }

    public function adherence(string $id): Psr7Response
    {
        return $this->response->json(($this->adherence)($this->who(), $this->uuid($id)));
    }

    public function exportFit(string $id): Psr7Response
    {
        $bytes = ($this->export)($this->who(), $this->uuid($id));

        return $this->response->raw($bytes)->withHeader('Content-Type', 'application/vnd.ant.fit')
            ->withHeader('Content-Disposition', 'attachment; filename="treino-' . $id . '.fit"');
    }

    private function who(): Principal
    {
        return $this->request->getAttribute(JwtAuthMiddleware::ATTRIBUTE);
    }

    /** @return array<string, mixed> */
    private function body(): array
    {
        $body = $this->request->getParsedBody();

        return is_array($body) ? $body : [];
    }

    private function uuid(string $id): string
    {
        if (!Uuid::isValid($id)) {
            throw new \App\Application\NotFound('recurso não encontrado');
        }

        return strtolower($id);
    }
}
