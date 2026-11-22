<?php

declare(strict_types=1);

namespace App\Application;

use RuntimeException;

/** Recurso inexistente ou de atleta que não consentiu: a resposta não distingue (ADR 0014). */
class NotFound extends RuntimeException
{
}
