<?php

declare(strict_types=1);

namespace App\Domain\Import;

/** Categorias de dado de saúde sobre as quais o atleta consente (ADR 0008: consentimento granular). */
enum DataCategory: string
{
    case Treinos = 'treinos';
    case FrequenciaCardiaca = 'frequencia_cardiaca';
    case Localizacao = 'localizacao';
    case Potencia = 'potencia';
}
