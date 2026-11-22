<?php

declare(strict_types=1);

namespace App\Infrastructure\Acl;

use App\Domain\Plan\Step;
use App\Domain\Plan\Workout;
use App\Domain\Shared\Modality;

/**
 * ACL de saída: Workout do domínio para arquivo FIT de treino estruturado (design doc 4.4).
 * Escrita binária direta do formato (cabeçalho de 14 bytes, mensagens de definição e dados, CRC-16),
 * sem nenhuma biblioteca de terceiros: não existe SDK FIT em PHP mantido. Alvos em % do limiar viram
 * alvos FIT quando o limiar é conhecido; senão o passo fica com alvo aberto.
 */
final class FitWorkoutTranslator
{
    private const PROFILE_VERSION = 2132;
    private const FIT_EPOCH_OFFSET = 631065600;
    private const MSG_FILE_ID = 0;
    private const MSG_WORKOUT = 26;
    private const MSG_WORKOUT_STEP = 27;
    private const CRC_TABLE = [0x0000, 0xCC01, 0xD801, 0x1400, 0xF001, 0x3C00, 0x2800, 0xE401, 0xA001, 0x6C00, 0x7800, 0xB401, 0x5000, 0x9C01, 0x8801, 0x4400];

    private const SPORT = [Modality::Swim->value => 5, Modality::Bike->value => 2, Modality::Run->value => 1, Modality::Strength->value => 10];
    private const INTENSITY = ['aquecimento' => 2, 'trabalho' => 0, 'recuperacao' => 4, 'desaquecimento' => 3];

    /** @param array<string, float> $thresholds ftp (W), pace_limiar (s/km), css (m/s) */
    public function translate(Workout $workout, array $thresholds = []): string
    {
        $data = $this->definition(self::MSG_FILE_ID, 0, [[0, 1, 0x00], [1, 2, 0x84], [2, 2, 0x84], [3, 4, 0x8C], [4, 4, 0x86]]);
        $data .= chr(0) . pack('C', 5) . pack('v', 255) . pack('v', 0) . pack('V', 1) . pack('V', max(0, $workout->date->getTimestamp() - self::FIT_EPOCH_OFFSET));

        $steps = $this->flatten($workout, $thresholds);
        $data .= $this->definition(self::MSG_WORKOUT, 1, [[4, 1, 0x00], [6, 2, 0x84], [8, 32, 0x07]]);
        $data .= chr(1) . pack('C', self::SPORT[$workout->modality->value]) . pack('v', count($steps)) . $this->text($workout->name, 32);

        $data .= $this->definition(self::MSG_WORKOUT_STEP, 2, [[254, 2, 0x84], [0, 16, 0x07], [1, 1, 0x00], [2, 4, 0x86], [3, 1, 0x00], [4, 4, 0x86], [5, 4, 0x86], [6, 4, 0x86], [7, 1, 0x00]]);
        foreach ($steps as $i => $s) {
            $data .= chr(2) . pack('v', $i) . $this->text($s['name'], 16) . pack('C', $s['durType']) . pack('V', $s['durValue'])
                . pack('C', $s['tgtType']) . pack('V', $s['tgtValue']) . pack('V', $s['low']) . pack('V', $s['high']) . pack('C', $s['intensity']);
        }

        $header = pack('CCvV', 14, 0x20, self::PROFILE_VERSION, strlen($data)) . '.FIT';
        $header .= pack('v', self::crc($header));

        return $header . $data . pack('v', self::crc($header . $data));
    }

    public static function crc(string $bytes, int $crc = 0): int
    {
        foreach (str_split($bytes) as $char) {
            $byte = ord($char);
            $tmp = self::CRC_TABLE[$crc & 0xF];
            $crc = (($crc >> 4) & 0x0FFF) ^ $tmp ^ self::CRC_TABLE[$byte & 0xF];
            $tmp = self::CRC_TABLE[$crc & 0xF];
            $crc = (($crc >> 4) & 0x0FFF) ^ $tmp ^ self::CRC_TABLE[($byte >> 4) & 0xF];
        }

        return $crc;
    }

    /**
     * @param array<string, float> $thresholds
     * @return list<array{name: string, durType: int, durValue: int, tgtType: int, tgtValue: int, low: int, high: int, intensity: int}>
     */
    private function flatten(Workout $workout, array $thresholds): array
    {
        $out = [];
        foreach ($workout->blocks as $block) {
            $first = count($out);
            foreach ($block->steps as $step) {
                $out[] = $this->step($step, $thresholds);
            }
            if ($block->repetitions > 1) {
                $out[] = ['name' => '', 'durType' => 6, 'durValue' => $first, 'tgtType' => 2, 'tgtValue' => $block->repetitions, 'low' => 0, 'high' => 0, 'intensity' => 0];
            }
        }

        return $out;
    }

    /**
     * @param array<string, float> $thresholds
     * @return array{name: string, durType: int, durValue: int, tgtType: int, tgtValue: int, low: int, high: int, intensity: int}
     */
    private function step(Step $step, array $thresholds): array
    {
        [$durType, $durValue] = match ($step->goalType) {
            'tempo' => [0, (int) round((float) $step->goalValue * 1000)],
            'distancia' => [1, (int) round((float) $step->goalValue * 100)],
            default => [5, 0],
        };
        [$tgtType, $low, $high] = $this->target($step, $thresholds);

        return ['name' => $step->type, 'durType' => $durType, 'durValue' => $durValue, 'tgtType' => $tgtType, 'tgtValue' => 0, 'low' => $low, 'high' => $high, 'intensity' => self::INTENSITY[$step->type] ?? 0];
    }

    /**
     * @param array<string, float> $thresholds
     * @return array{0: int, 1: int, 2: int}
     */
    private function target(Step $step, array $thresholds): array
    {
        $open = [2, 0, 0];
        if ($step->targetType === null) {
            return $open;
        }
        $low = (float) $step->targetLow;
        $high = (float) $step->targetHigh;

        return match ($step->targetType) {
            'potencia_pct_ftp' => [4, (int) round($low), (int) round($high)], // valores até 1000 são % do FTP no FIT
            'fc_pct_max' => [1, (int) round($low), (int) round($high)],
            'pace_pct_limiar' => isset($thresholds['pace_limiar']) && $thresholds['pace_limiar'] > 0
                ? [0, $this->speed(1000 / $thresholds['pace_limiar'], $low), $this->speed(1000 / $thresholds['pace_limiar'], $high)]
                : $open,
            'css_pct' => isset($thresholds['css']) && $thresholds['css'] > 0
                ? [0, $this->speed($thresholds['css'], $low), $this->speed($thresholds['css'], $high)]
                : $open,
            default => $open,
        };
    }

    /** Velocidade FIT em mm/s. */
    private function speed(float $metersPerSecond, float $percent): int
    {
        return (int) round($metersPerSecond * $percent / 100 * 1000);
    }

    /** @param list<array{0: int, 1: int, 2: int}> $fields número, tamanho, tipo base */
    private function definition(int $globalNumber, int $local, array $fields): string
    {
        $out = pack('CCC', 0x40 | $local, 0, 0) . pack('v', $globalNumber) . pack('C', count($fields));
        foreach ($fields as [$num, $size, $type]) {
            $out .= pack('CCC', $num, $size, $type);
        }

        return $out;
    }

    private function text(string $value, int $size): string
    {
        return str_pad(substr($value, 0, $size - 1), $size, "\0");
    }
}
