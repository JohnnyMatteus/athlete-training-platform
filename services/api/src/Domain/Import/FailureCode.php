<?php

declare(strict_types=1);

namespace App\Domain\Import;

enum FailureCode: string
{
    case Sha256Divergente = 'sha256_divergente';
    case ArquivoInvalido = 'arquivo_invalido';
    case FormatoNaoSuportado = 'formato_nao_suportado';
    case FalhaInterna = 'falha_interna';
}
