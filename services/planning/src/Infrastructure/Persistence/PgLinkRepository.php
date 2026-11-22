<?php

declare(strict_types=1);

namespace App\Infrastructure\Persistence;

use App\Application\Port\LinkRepository;
use App\Domain\Link\Link;
use Hyperf\DbConnection\Db;

final class PgLinkRepository implements LinkRepository
{
    public function save(Link $link): void
    {
        Db::statement(
            'INSERT INTO vinculo_treinador (id, atleta_id, treinador_id, status, categorias) VALUES (?, ?, ?, ?, ?::text[])
             ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, atualizado_em = now()',
            [$link->id, $link->athleteId, $link->coachId, $link->status, '{' . implode(',', $link->categories) . '}'],
        );
    }

    public function find(string $id): ?Link
    {
        $row = Db::table('vinculo_treinador')->where('id', $id)->first();

        return $row === null ? null : $this->hydrate($row);
    }

    public function listFor(string $userId): array
    {
        return Db::table('vinculo_treinador')->where('atleta_id', $userId)->orWhere('treinador_id', $userId)
            ->orderBy('atualizado_em', 'desc')->get()->map(fn (object $r): Link => $this->hydrate($r))->all();
    }

    public function hasActive(string $athleteId, string $coachId, string $category): bool
    {
        return Db::table('vinculo_treinador')
            ->where('atleta_id', $athleteId)->where('treinador_id', $coachId)->where('status', 'ativo')
            ->whereRaw('? = ANY(categorias)', [$category])->exists();
    }

    private function hydrate(object $r): Link
    {
        $categories = is_string($r->categorias) ? explode(',', trim($r->categorias, '{}')) : (array) $r->categorias;

        return new Link((string) $r->id, (string) $r->atleta_id, (string) $r->treinador_id, (string) $r->status, array_values(array_filter($categories, static fn (string $c): bool => $c !== '')));
    }
}
