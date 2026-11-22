<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Application\Import\CreateImport;
use App\Application\Import\GetImport;
use App\Application\Import\NotFound;
use App\Domain\Import\AthleteId;
use App\Domain\Import\ImportId;
use App\Domain\Import\ImportStatus;
use App\Domain\Import\ValidationFailed;
use DateInterval;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;
use Tests\Support\FixedClock;
use Tests\Support\InMemoryBatches;
use Tests\Support\Samples;

final class CreateImportTest extends TestCase
{
    private InMemoryBatches $batches;
    private CreateImport $create;

    protected function setUp(): void
    {
        $this->batches = new InMemoryBatches();
        $this->create = new CreateImport(
            $this->batches,
            new FixedClock(),
            'http://tus.test/files/',
            2 * 1024 ** 3,
            new DateInterval('PT24H'),
        );
    }

    public function testValidMetadataCreatesBatchAwaitingUpload(): void
    {
        $result = ($this->create)(new AthleteId(Samples::ATHLETE), Samples::input());

        self::assertSame('http://tus.test/files/', $result['upload_url']);
        $batch = $this->batches->find(new ImportId($result['import_id']));
        self::assertNotNull($batch);
        self::assertSame(ImportStatus::AguardandoUpload, $batch->status());
    }

    public function testConsentIsRecordedWithTermVersion(): void
    {
        $result = ($this->create)(new AthleteId(Samples::ATHLETE), Samples::input());

        $consent = $this->batches->find(new ImportId($result['import_id']))->consent;
        self::assertSame('2026-11', $consent->termVersion);
        self::assertSame(['treinos', 'frequencia_cardiaca'], $consent->categoryValues());
    }

    public function testSameFileSha256CreatesNewBatch(): void
    {
        $athlete = new AthleteId(Samples::ATHLETE);
        $a = ($this->create)($athlete, Samples::input());
        $b = ($this->create)($athlete, Samples::input());

        self::assertNotSame($a['import_id'], $b['import_id']);
        self::assertCount(2, $this->batches->items);
    }

    /** @return iterable<string, array{array<string, mixed>, string}> */
    public static function invalidInputs(): iterable
    {
        yield 'sem sha256' => [['sha256' => null], 'sha256'];
        yield 'sha256 curto' => [['sha256' => 'abc'], 'sha256'];
        yield 'tipo fora da lista' => [['tipo' => 'exe', 'nome_arquivo' => 'a.exe'], 'tipo'];
        yield 'extensão diferente do tipo' => [['tipo' => 'fit', 'nome_arquivo' => 'a.gpx'], 'nome_arquivo'];
        yield 'nome com caminho' => [['nome_arquivo' => '../export.zip'], 'nome_arquivo'];
        yield 'tamanho zero' => [['tamanho_bytes' => 0], 'tamanho_bytes'];
        yield 'tamanho acima do limite' => [['tamanho_bytes' => 3 * 1024 ** 3], 'tamanho_bytes'];
        yield 'sem consentimento' => [['consentimento' => null], 'consentimento'];
        yield 'consentimento recusado' => [['consentimento' => ['aceito' => false, 'versao_termo' => '1', 'categorias' => ['treinos']]], 'consentimento'];
        yield 'categoria desconhecida' => [['consentimento' => ['aceito' => true, 'versao_termo' => '1', 'categorias' => ['dna']]], 'consentimento'];
    }

    /** @param array<string, mixed> $override */
    #[DataProvider('invalidInputs')]
    public function testInvalidInputIsRejectedWithFieldError(array $override, string $field): void
    {
        try {
            ($this->create)(new AthleteId(Samples::ATHLETE), Samples::input($override));
            self::fail('esperava ValidationFailed');
        } catch (ValidationFailed $e) {
            self::assertArrayHasKey($field, $e->errors);
            self::assertSame([], $this->batches->items);
        }
    }

    public function testAllFieldErrorsAreReportedTogether(): void
    {
        try {
            ($this->create)(new AthleteId(Samples::ATHLETE), []);
            self::fail('esperava ValidationFailed');
        } catch (ValidationFailed $e) {
            self::assertEqualsCanonicalizing(
                ['tipo', 'sha256', 'consentimento', 'nome_arquivo', 'tamanho_bytes'],
                array_keys($e->errors),
            );
        }
    }

    public function testStatusQueryIsScopedToTheOwner(): void
    {
        $result = ($this->create)(new AthleteId(Samples::ATHLETE), Samples::input());
        $get = new GetImport($this->batches);
        $id = new ImportId($result['import_id']);

        self::assertSame('aguardando_upload', $get(new AthleteId(Samples::ATHLETE), $id)['status']);

        $this->expectException(NotFound::class);
        $get(new AthleteId('0190a1b2-c3d4-7e5f-8a9b-ffffffffffff'), $id);
    }

    #[DataProvider('fileTypes')]
    public function testAcceptsEachSupportedFileType(string $type): void
    {
        $result = ($this->create)(
            new AthleteId(Samples::ATHLETE),
            Samples::input(['tipo' => $type, 'nome_arquivo' => "treino.$type"]),
        );

        self::assertSame($type, $this->batches->find(new ImportId($result['import_id']))->type->value);
    }

    /** @return iterable<string, array{string}> */
    public static function fileTypes(): iterable
    {
        foreach (['zip', 'fit', 'tcx', 'gpx'] as $t) {
            yield $t => [$t];
        }
    }
}
