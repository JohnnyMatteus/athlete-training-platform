<?php

declare(strict_types=1);

namespace App\Domain\Import;

enum ImportStatus: string
{
    case AguardandoUpload = 'aguardando_upload';
    case Recebido = 'recebido';
    case Processando = 'processando';
    case Concluido = 'concluido';
    case Falhou = 'falhou';
}
