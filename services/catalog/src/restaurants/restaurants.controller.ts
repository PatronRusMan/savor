import { Body, Controller, ForbiddenException, Get, Headers, Param, Patch, Post, Query } from "@nestjs/common";
import { RestaurantsService } from "./restaurants.service";

@Controller()
export class RestaurantsController {
  constructor(private readonly svc: RestaurantsService) {}

  @Get("/restaurants")
  list(@Query("q") q?: string, @Query("cuisine") cuisine?: string) {
    return this.svc.list(q, cuisine);
  }

  @Get("/restaurants/mine")
  mine(@Headers("x-user-id") userId: string) {
    return this.svc.mine(userId);
  }

  @Get("/restaurants/:id")
  byId(@Param("id") id: string) {
    return this.svc.byId(id);
  }

  @Get("/restaurants/:id/menu")
  menu(@Param("id") id: string) {
    return this.svc.menu(id);
  }

  @Post("/restaurants")
  create(
    @Headers("x-user-id") userId: string,
    @Headers("x-user-role") role: string,
    @Body() body: { name: string; description: string; cuisine: string; address: string; etaMinutes?: number; imageUrl?: string },
  ) {
    if (role !== "restaurant") throw new ForbiddenException({ error: { code: "forbidden", message: "restaurant role required" } });
    return this.svc.create(userId, body);
  }

  @Patch("/restaurants/:id")
  update(
    @Param("id") id: string,
    @Headers("x-user-id") userId: string,
    @Headers("x-user-role") role: string,
    @Body() body: Record<string, unknown>,
  ) {
    return this.svc.update(id, userId, role, body);
  }

  @Post("/restaurants/:id/dishes")
  addDish(
    @Param("id") id: string,
    @Headers("x-user-id") userId: string,
    @Body() body: { name: string; description?: string; priceCents: number; category?: string },
  ) {
    return this.svc.addDish(id, userId, body);
  }
}
