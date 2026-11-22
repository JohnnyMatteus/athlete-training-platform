import { z } from 'zod';

// Uma URL por serviço, como em contracts/openapi (portas 9501, 8120, 8121, 8122).
const schema = z.object({
  VITE_API_URL: z.url().default('http://localhost:9501'),
  VITE_PLANEJAMENTO_URL: z.url().default('http://localhost:8120'),
  VITE_METRICAS_URL: z.url().default('http://localhost:8121'),
  VITE_ATIVIDADES_URL: z.url().default('http://localhost:8122'),
  VITE_COACH_URL: z.url().default('http://localhost:8123'),
});

export const env = schema.parse(import.meta.env);
