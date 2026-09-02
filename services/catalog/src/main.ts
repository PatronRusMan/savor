import "reflect-metadata";
import { NestFactory } from "@nestjs/core";
import { AppModule } from "./app.module";
import { json } from "express";

async function bootstrap() {
  const app = await NestFactory.create(AppModule, { logger: ["error", "warn", "log"] });
  app.use(json({ limit: "1mb" }));
  const port = Number(process.env.PORT ?? 8002);
  await app.listen(port, "0.0.0.0");
}

bootstrap();
