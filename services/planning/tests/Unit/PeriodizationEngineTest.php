<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Domain\Plan\Constraints;
use App\Domain\Plan\ConstraintValidator;
use App\Domain\Plan\Phase;
use App\Domain\Plan\Plan;
use App\Domain\Plan\Violation;
use App\Domain\Shared\Intensity;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;
use Tests\Support\Samples;

final class PeriodizationEngineTest extends TestCase
{
    public function testSameInputsGiveSamePlan(): void
    {
        self::assertSame(Samples::plan()->toArray(), Samples::plan()->toArray());
    }

    /** @return iterable<string, array{float, string, string}> */
    public static function scenarios(): iterable
    {
        yield 'meio ironman, CTL moderado' => [38.5, 'meio_ironman', '2027-06-06'];
        yield 'iniciante sem histórico' => [0.0, 'triatlo_olimpico', '2027-03-07'];
        yield 'atleta com CTL alto' => [85.0, 'ironman', '2027-09-05'];
        yield 'prova em 3 semanas' => [20.0, 'triatlo_sprint', '2026-12-20'];
    }

    #[DataProvider('scenarios')]
    public function testGeneratedPlanNeverViolatesMandatoryRules(float $ctl, string $race, string $raceDate): void
    {
        $plan = Samples::plan($ctl, $race, $raceDate);

        self::assertFalse(ConstraintValidator::hasMandatory((new ConstraintValidator())->validate($plan)));
        self::assertNotEmpty($plan->workouts);
        foreach ($plan->workouts as $w) {
            self::assertLessThan($plan->raceDate, $w->date);
            self::assertEqualsWithDelta($w->durationS, array_sum(array_map(static fn ($b) => $b->seconds(), $w->blocks)), 0.5, "{$w->id} blocos somam a duração");
        }
    }

    public function testPhasesFollowDesignAndDeloadEveryFourthWeek(): void
    {
        $plan = Samples::plan();
        $phases = array_map(static fn (array $w): Phase => $w['fase'], $plan->weeks);

        self::assertSame(Phase::Base, $phases[0]);
        self::assertSame(Phase::Taper, $phases[array_key_last($phases)]);
        self::assertSame(2, count(array_filter($phases, static fn (Phase $p): bool => $p === Phase::Taper)));
        self::assertSame(3, count(array_filter($phases, static fn (Phase $p): bool => $p === Phase::Peak)));
        self::assertTrue($plan->weeks[3]['descarga']);
        self::assertFalse($plan->weeks[2]['descarga']);
        self::assertLessThan($this->weekTss($plan, 3), $this->weekTss($plan, 4), 'semana de descarga tem menos carga que a anterior');
    }

    public function testCtlRampStaysWithinConfiguredLimit(): void
    {
        $plan = Samples::plan(30.0, 'meio_ironman', '2027-06-06', '2026-11-30', new Constraints(maxCtlRamp: 3.0));
        $violations = (new ConstraintValidator())->validate($plan);

        self::assertSame([], array_filter($violations, static fn (Violation $v): bool => $v->rule === Violation::RAMP));
    }

    public function testPolarizedDistributionKeepsMostTimeInLowIntensity(): void
    {
        $plan = Samples::plan();
        $low = $total = 0;
        foreach ($plan->workouts as $w) {
            $total += $w->durationS;
            $low += $w->intensity === Intensity::Low ? $w->durationS : 0;
        }

        self::assertGreaterThanOrEqual(0.7, $low / $total);
    }

    private function weekTss(Plan $plan, int $week): float
    {
        return array_sum(array_map(static fn ($w): float => $w->tss, $plan->workoutsOfWeek($week)));
    }
}
