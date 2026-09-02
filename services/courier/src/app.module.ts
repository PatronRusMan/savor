import { Module } from "@nestjs/common";
import { PrismaService } from "./prisma/prisma.service";
import { MqService } from "./mq/mq.service";
import { HealthController } from "./health/health.controller";
import { CourierController } from "./courier/courier.controller";
import { CourierService } from "./courier/courier.service";

@Module({
  controllers: [HealthController, CourierController],
  providers: [PrismaService, MqService, CourierService],
})
export class AppModule {}
