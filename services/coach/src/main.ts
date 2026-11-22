import { compose } from "./compose.js";
import { loadConfig } from "./config.js";

const cfg = loadConfig(process.env);
const { app, redis } = compose(cfg);

const stop = async () => {
  await app.close();
  redis.disconnect();
};
process.once("SIGTERM", () => void stop());
process.once("SIGINT", () => void stop());

await app.listen({ host: "0.0.0.0", port: cfg.port });
