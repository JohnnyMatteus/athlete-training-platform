<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Domain\Plan\Adjustment;
use App\Domain\Plan\Availability;
use App\Domain\Plan\Constraints;
use App\Domain\Plan\ConstraintValidator;
use App\Domain\Plan\Phase;
use App\Domain\Plan\Plan;
use App\Domain\Plan\PlanStatus;
use App\Domain\Plan\Violation;
use App\Domain\Plan\WorkoutBuilder;
use App\Domain\Shared\Intensity;
use App\Domain\Shared\Modality;
use InvalidArgumentException;
use PHPUnit\Framework\TestCase;
use Tests\Support\Samples;

/** Regras do design doc 7.3, cada uma com valores calculados à mão. */
final class ConstraintValidatorTest extends TestCase
{
    private WorkoutBuilder $builder;

    protected function setUp(): void
    {
        $this->builder = new WorkoutBuilder();
    }

    /** Uma semana (segunda 2026-11-30) com disponibilidade de 8 h por dia, para isolar a regra testada. */
    private function plan(array $sessions, float $ctl = 50.0, ?Constraints $c = null): Plan
    {
        $availability = array_map(static fn (int $d): Availability => new Availability($d, 480, [Modality::Run, Modality::Strength, Modality::Bike]), range(1, 7));
        $workouts = [];
        foreach ($sessions as $i => [$offset, $modality, $intensity, $minutes]) {
            $workouts[] = $this->builder->build(sprintf('00000000-0000-4000-8000-%012d', $i), Samples::date('2026-11-30')->modify("+{$offset} days"), $modality, $intensity, $minutes * 60);
        }

        return (new Plan(
            Samples::PLAN,
            Samples::ATHLETE,
            PlanStatus::Active,
            1,
            'meio_ironman',
            Samples::date('2027-06-06'),
            Samples::date('2026-11-30'),
            false,
            null,
            $ctl,
            $ctl,
            $c ?? new Constraints(),
            $availability,
            [['fase' => Phase::Build, 'descarga' => false]],
            [],
        ))->with($workouts);
    }

    /** @return list<string> */
    private function rules(Plan $plan): array
    {
        return array_values(array_unique(array_map(static fn (Violation $v): string => $v->rule, (new ConstraintValidator())->validate($plan))));
    }

    public function testRampAboveLimit(): void
    {
        // 6 dias de 150 TSS partindo de CTL 50: CTL sobe de 50 para 50 + 6*(150-50)/42 ~ 64 (> 5)
        $plan = $this->plan(array_map(static fn (int $d): array => [$d - 1, Modality::Bike, Intensity::Moderate, 280], range(1, 6)));

        self::assertContains(Violation::RAMP, $this->rules($plan));
    }

    public function testRampWithinLimitPasses(): void
    {
        $plan = $this->plan([[0, Modality::Run, Intensity::Low, 60], [2, Modality::Run, Intensity::Low, 60]], 50.0);

        self::assertNotContains(Violation::RAMP, $this->rules($plan));
    }

    public function testTsbBelowMinimumBlocksHardSession(): void
    {
        // CTL 20, ATL 60: TSB = -40 antes de uma sessão alta
        $plan = $this->plan([[0, Modality::Run, Intensity::High, 60]]);
        $plan = new Plan(...[...get_object_vars($plan), 'initialCtl' => 20.0, 'initialAtl' => 60.0]);

        $v = array_values(array_filter((new ConstraintValidator())->validate($plan), static fn (Violation $v): bool => $v->rule === Violation::TSB));
        self::assertCount(1, $v);
        self::assertSame(-40.0, $v[0]->value);
        self::assertSame(-25.0, $v[0]->limit);
    }

    public function testTsbIsNotCheckedForEasySessions(): void
    {
        $plan = $this->plan([[0, Modality::Run, Intensity::Low, 60]]);
        $plan = new Plan(...[...get_object_vars($plan), 'initialCtl' => 20.0, 'initialAtl' => 60.0]);

        self::assertNotContains(Violation::TSB, $this->rules($plan));
    }

    public function testThreeConsecutiveHardDaysViolateTwoDayLimit(): void
    {
        $three = $this->plan([[0, Modality::Run, Intensity::High, 30], [1, Modality::Bike, Intensity::High, 30], [2, Modality::Run, Intensity::High, 30]]);
        $two = $this->plan([[0, Modality::Run, Intensity::High, 30], [1, Modality::Bike, Intensity::High, 30]]);

        self::assertContains(Violation::CONSECUTIVE_HARD, $this->rules($three));
        self::assertNotContains(Violation::CONSECUTIVE_HARD, $this->rules($two));
    }

    public function testWeekWithoutRestDayViolates(): void
    {
        $seven = $this->plan(array_map(static fn (int $i): array => [$i - 1, Modality::Run, Intensity::Low, 20], range(1, 7)));
        $six = $this->plan(array_map(static fn (int $i): array => [$i - 1, Modality::Run, Intensity::Low, 20], range(1, 6)));

        self::assertContains(Violation::NO_REST, $this->rules($seven));
        self::assertNotContains(Violation::NO_REST, $this->rules($six));
    }

    public function testVolumeAboveDeclaredAvailability(): void
    {
        $plan = $this->plan([[0, Modality::Run, Intensity::Low, 60]]);
        $limited = new Plan(...[...get_object_vars($plan), 'availability' => [new Availability(1, 45, [Modality::Run])]]);

        $v = array_values(array_filter((new ConstraintValidator())->validate($limited), static fn (Violation $v): bool => $v->rule === Violation::VOLUME));
        self::assertNotEmpty($v);
        self::assertSame(60.0, $v[0]->value);
        self::assertSame(45.0, $v[0]->limit);
    }

    public function testStrengthWithRunThresholdIsOnlyPreferred(): void
    {
        $plan = $this->plan([[0, Modality::Strength, Intensity::Low, 45], [0, Modality::Run, Intensity::Moderate, 60]]);
        $v = array_values(array_filter((new ConstraintValidator())->validate($plan), static fn (Violation $v): bool => $v->rule === Violation::STRENGTH_RUN));

        self::assertCount(1, $v);
        self::assertFalse($v[0]->mandatory);
        self::assertFalse(ConstraintValidator::hasMandatory($v));
    }

    public function testPainViolationIsMandatoryAndAdvisesProfessional(): void
    {
        $v = ConstraintValidator::painBlocked();

        self::assertTrue($v->mandatory);
        self::assertStringContainsString('profissional', $v->message);
    }

    public function testAdjustmentRebuildsWorkoutAndBumpsVersion(): void
    {
        $plan = Samples::plan();
        $target = $plan->workouts[0];
        [$next, $touched] = (new Adjustment())->apply($plan, [['treino_id' => $target->id, 'acao' => 'alterar_intensidade', 'nova_intensidade' => 'alta']]);

        self::assertSame(2, $next->version);
        self::assertSame([$target->id], $touched);
        self::assertSame(Intensity::High, $next->workout($target->id)?->intensity);
        self::assertSame(1, $plan->version, 'o plano original não muda');
    }

    public function testAdjustmentRejectsUnknownWorkoutAndDateOutsidePlan(): void
    {
        $plan = Samples::plan();
        $this->expectException(InvalidArgumentException::class);
        (new Adjustment())->apply($plan, [['treino_id' => '00000000-0000-4000-8000-000000000000', 'acao' => 'remover']]);
    }

    public function testMovingIntoAnUnavailableDayIsCaughtByValidator(): void
    {
        $plan = Samples::plan();
        $w = $plan->workouts[0];
        // sexta-feira (5) não tem disponibilidade
        $friday = Samples::date('2026-12-04');
        [$next] = (new Adjustment())->apply($plan, [['treino_id' => $w->id, 'acao' => 'mover', 'nova_data' => $friday->format('Y-m-d')]]);

        self::assertContains(Violation::VOLUME, $this->rules($next));
    }
}
