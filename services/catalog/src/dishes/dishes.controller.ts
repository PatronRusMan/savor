import { Body, Controller, Delete, Get, Headers, Param, Patch } from "@nestjs/common";
import { RestaurantsService } from "../restaurants/restaurants.service";

@Controller()
export class DishesController {
  constructor(private readonly svc: RestaurantsService) {}

  @Get("/dishes/:id")
  dish(@Param("id") id: string) {
    return this.svc.dish(id);
  }

  @Patch("/dishes/:id")
  patch(@Param("id") id: string, @Headers("x-user-id") userId: string, @Body() body: Record<string, unknown>) {
    return this.svc.patchDish(id, userId, body);
  }

  @Delete("/dishes/:id")
  remove(@Param("id") id: string, @Headers("x-user-id") userId: string) {
    return this.svc.deleteDish(id, userId);
  }
}
