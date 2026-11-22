<?php

declare(strict_types=1);

namespace App\Application\Link;

use App\Application\Conflict;
use App\Application\Forbidden;
use App\Application\NotFound;
use App\Application\Port\EventOutbox;
use App\Application\Port\LinkRepository;
use App\Application\Port\Transaction;
use App\Application\ValidationFailed;
use App\Domain\Link\Link;
use App\Domain\Shared\Principal;
use DomainException;
use Psr\Clock\ClockInterface;
use Ramsey\Uuid\Uuid;

/** Vínculo consentido (ADR 0014): convite do atleta, aceite do treinador, revogação de qualquer um. */
final readonly class LinkService
{
    public function __construct(
        private LinkRepository $links,
        private EventOutbox $outbox,
        private Transaction $tx,
        private ClockInterface $clock,
    ) {
    }

    /** @param list<mixed> $categories */
    public function invite(Principal $who, string $coachId, array $categories): Link
    {
        if (!$who->isAthlete()) {
            throw new Forbidden('só o atleta convida');
        }
        $errors = [];
        if (!Uuid::isValid($coachId) || $coachId === $who->id) {
            $errors['treinador_id'] = 'Informe o UUID de outro usuário.';
        }
        if ($categories === [] || array_diff($categories, Link::CATEGORIES) !== []) {
            $errors['categorias'] = 'Use ' . implode(', ', Link::CATEGORIES) . '.';
        }
        if ($errors !== []) {
            throw new ValidationFailed($errors);
        }
        $link = Link::invite(Uuid::uuid7()->toString(), $who->id, $coachId, array_values(array_unique($categories)));
        $this->links->save($link);

        return $link;
    }

    public function accept(Principal $who, string $id): Link
    {
        $link = $this->links->find($id);
        if ($link === null || $link->coachId !== $who->id) {
            throw new NotFound('vínculo não encontrado');
        }
        if (!$who->isCoach()) {
            throw new Forbidden('só treinador aceita');
        }
        try {
            $active = $link->accept();
        } catch (DomainException $e) {
            throw new Conflict($e->getMessage());
        }
        $this->tx->run(function () use ($active, $link): void {
            // Condicional ao status lido: uma revogação concorrente vence e o aceite vira 409 (revogado é terminal).
            if (!$this->links->transition($active, $link->status)) {
                throw new Conflict('o vínculo mudou durante o aceite');
            }
            $this->outbox->append($active->consentedEvent($this->clock->now()));
        });

        return $active;
    }

    public function revoke(Principal $who, string $id): void
    {
        for ($attempt = 0; $attempt < 3; ++$attempt) {
            $link = $this->links->find($id);
            if ($link === null || ($link->athleteId !== $who->id && $link->coachId !== $who->id)) {
                throw new NotFound('vínculo não encontrado');
            }
            if ($link->status === 'revogado') {
                return;
            }
            $by = $link->athleteId === $who->id ? 'atleta' : 'treinador';
            $revoked = $link->revoke();
            $wasActive = $link->status === 'ativo';
            $done = $this->tx->run(function () use ($link, $revoked, $by, $wasActive): bool {
                if (!$this->links->transition($revoked, $link->status)) {
                    return false;
                }
                if ($wasActive) { // convite nunca aceito não chegou a ser publicado
                    $this->outbox->append($revoked->revokedEvent($by, $this->clock->now()));
                }

                return true;
            });
            if ($done) {
                return;
            }
        }

        throw new Conflict('o vínculo mudou durante a revogação; tente de novo');
    }

    /** @return list<Link> */
    public function list(Principal $who): array
    {
        return $this->links->listFor($who->id);
    }
}
