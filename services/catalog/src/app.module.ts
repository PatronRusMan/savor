import { Module } from "@nestjs/common";
import { PrismaService } from "./prisma/prisma.service";
import { RedisService } from "./redis/redis.service";
import { HealthController } from "./health/health.controller";
import { RestaurantsController } from "./restaurants/restaurants.controller";
import { RestaurantsService } from "./restaurants/restaurants.service";
import { DishesController } from "./dishes/dishes.controller";
import { SeedService } from "./restaurants/seed.service";

@Module({
  controllers: [HealthController, RestaurantsController, DishesController],
  providers: [PrismaService, RedisService, RestaurantsService, SeedService],
})
export class AppModule {}
