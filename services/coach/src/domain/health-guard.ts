/** Pain, injury and symptoms get no prescription (design doc 7.3/7.4, ADR 0005). Code, not prompt. */

export const STANDARD_HEALTH_RESPONSE =
  "Você mencionou dor, lesão ou um sintoma. Não consigo orientar nem ajustar treino nessa situação. " +
  "Procure um médico ou fisioterapeuta antes de seguir com a intensidade planejada. " +
  "Nenhuma alteração foi feita no seu plano.";

// A trailing "*" marks a stem; everything else must match a whole word ("fever" must not catch "fevereiro").
const TERMS = [
  "dor", "dores", "doi", "doem", "doendo", "dolorid*", "machuc*", "lesao", "lesoes", "lesionei", "lesionad*",
  "fratura*", "tendinite*", "torcao", "torci", "inchaco", "inchad*", "contratura*", "distensao", "rompi", "ruptura*",
  "tontura*", "tonto", "tonta", "desmai*", "palpitac*", "arritmia*", "falta de ar", "formigamento", "dormencia",
  "enjoo", "nausea*", "febre", "sintoma*", "mal estar", "caimbra*", "caibra*", "pontada*", "sangr*",
  "pain", "painful", "injur*", "hurt", "hurts", "sprain*", "strain", "dizzy", "faint", "fever", "symptom*",
  "chest tight", "numb", "tingling", "swollen",
];
const NEGATORS = new Set(["sem", "nenhuma", "nenhum", "no", "without", "not", "nao", "nunca", "never"]);

const FILLERS = new Set(["tenho", "sinto", "senti", "tive", "ha", "tem", "have", "feel", "any", "a", "nenhuma"]);

const normalize = (s: string): string =>
  s
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase();

/** True when the text mentions pain, injury or a symptom that is not plainly negated. */
export function mentionsHealthIssue(text: string): boolean {
  const norm = normalize(text);
  const words = norm.split(/[^a-z0-9]+/).filter(Boolean);
  for (const term of TERMS) {
    const first = term.split(" ")[0] ?? term;
    const termWords = term.split(" ").length;
    for (let i = 0; i < words.length; i++) {
      const w = words[i] ?? "";
      const stem = term.endsWith("*");
      const hit = termWords === 1 ? (stem ? w.startsWith(term.slice(0, -1)) : w === term) : w === first && words.slice(i, i + termWords).join(" ") === term;
      if (!hit) continue;
      // "sem dor" and "no pain" do not count; "sem dor, mas meu joelho doi" still does through "doi".
      const prev = words[i - 1] ?? "";
      const prev2 = words[i - 2] ?? "";
      if (NEGATORS.has(prev) || (NEGATORS.has(prev2) && FILLERS.has(prev))) continue;
      return true;
    }
  }
  return false;
}

// Medical advice the model must never emit: diagnosis, medication, treatment.
const MEDICAL_ADVICE = [
  /\bdiagnost/, /\bdiagnos/, /\bvoce tem (uma |um )?(tendinite|fratura|lesao|infeccao|artrose|bursite|sindrome)/,
  /\b(ibuprofeno|paracetamol|dipirona|diclofenaco|naproxeno|aspirina|anti-?inflamatori|analgesic|corticoide|medicament|remedio)/,
  /\b(you have|you are suffering from) (a |an )?(tendinitis|fracture|injury|infection|syndrome)/,
  /\b(take|tome|tomar)\b.{0,30}\b(mg|comprimid|pill|dose|dosagem)/,
  /\b(gelo|compressa|imobiliz|infiltracao|cirurgia|fisioterapia em casa)/,
  /\b(treat|treatment|tratamento|tratar)\b/,
];

export function containsMedicalAdvice(text: string): boolean {
  const norm = normalize(text);
  return MEDICAL_ADVICE.some((re) => re.test(norm));
}
