<?php

declare(strict_types=1);

namespace App\Domain\Import;

enum FileType: string
{
    case Zip = 'zip';
    case Fit = 'fit';
    case Tcx = 'tcx';
    case Gpx = 'gpx';

    public static function fromInput(mixed $value): self
    {
        $type = is_string($value) ? self::tryFrom(strtolower($value)) : null;

        return $type ?? throw ValidationFailed::field('tipo', 'deve ser um de: zip, fit, tcx, gpx');
    }

    public function matchesFileName(string $name): bool
    {
        return strtolower(pathinfo($name, PATHINFO_EXTENSION)) === $this->value;
    }
}
