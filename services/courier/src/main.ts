import "reflect-metadata";
import { NestFactory } from "@nestjs/core";
import { AppModule } from "./app.module";

async function bootstrap() {
  const app = await NestFactory.create(AppModule, { logger: ["error", "warn", "log"] });
  await app.listen(Number(process.env.PORT ?? 8005), "0.0.0.0");
}
bootstrap();
